package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	adminsearch "github.com/ardaozyildirim/hydrogo/internal/search"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
	trashstore "github.com/ardaozyildirim/hydrogo/internal/trash/sqlstore"
	trashtransport "github.com/ardaozyildirim/hydrogo/internal/trash/transport"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// collectionRoutes registers the YAML content types under /admin/resources
// and /api/v1/content.
func (a *serveApp) collectionRoutes() ([]router.Route, error) {
	ctx, cfg, db := a.ctx, a.cfg, a.db
	if !a.modules["collections"] {
		return nil, nil
	}
	collectionDB := collectionstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope}
	collectionService := collections.Service{Repository: collectionDB}
	contentRoutes, err := contentTypeRoutes(ctx, cfg.ContentTypesDir, collectionService, a.tenantScopes,
		contenttypes.NewLocales(cfg.I18N.Enabled, cfg.I18N.DefaultLocale, cfg.I18N.Locales))
	if err != nil {
		return nil, err
	}
	// contentTypeRoutes has validated the directory; list it for the sidebar.
	definitions, err := contenttypes.LoadDir(cfg.ContentTypesDir)
	if err != nil {
		return nil, err
	}
	a.contentTypeLinks = make([]adminstudio.ContentTypeLink, 0, len(definitions))
	for _, definition := range definitions {
		a.contentTypeLinks = append(a.contentTypeLinks, adminstudio.ContentTypeLink{ID: definition.ID, Label: definition.Label})
	}
	return contentRoutes, nil
}

// trashRoutes registers the recycle bin for the enabled content kinds and
// clears public caches after page or news restores and purges.
func (a *serveApp) trashRoutes() []router.Route {
	cfg, db, modules := a.cfg, a.db, a.modules
	requireTenantScope := a.requireTenantScope
	publicPageCache, publicHTMLCache, staticCacheDir := a.publicPageCache, a.publicHTMLCache, a.staticCacheDir
	trashKinds := map[trash.Kind]bool{trash.Pages: modules["pages"], trash.News: modules["news"], trash.Submissions: modules["site"], trash.Media: modules["media"]}
	if !(trashKinds[trash.Pages] || trashKinds[trash.News] || trashKinds[trash.Submissions] || trashKinds[trash.Media]) {
		return nil
	}
	trashRepository := trashstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Enabled: trashKinds}
	trashHandler := trashtransport.Handler{Service: trash.Service{
		Repository:   trashRepository,
		Transactions: a.transactioner,
		Audit:        auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
	}}
	trashHandler.AfterMutation = func(_ context.Context, kind trash.Kind, _ string) {
		if kind == trash.Pages || kind == trash.News {
			if publicPageCache != nil {
				publicPageCache.Clear()
			}
			if publicHTMLCache != nil {
				publicHTMLCache.Clear()
			}
			if staticCacheDir != "" {
				if err := server.PurgeStaticDiskCache(staticCacheDir); err != nil {
					slog.Warn("trash cache invalidation failed", "error", err)
				}
			}
		}
	}
	return trashHandler.Routes()
}

func (a *serveApp) searchRoutes() []router.Route {
	cfg, db, modules := a.cfg, a.db, a.modules
	requireTenantScope := a.requireTenantScope
	if !modules["search"] {
		return nil
	}
	var routes []router.Route
	routes = append(routes, router.Route{Pattern: "GET /api/v1/search/autocomplete", Access: router.Public, Handler: adminsearch.AutocompleteHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Modules: modules}})
	autocomplete := adminsearch.AutocompleteHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Modules: modules}
	routes = append(routes, router.Route{Pattern: "GET /search", Access: router.Public, Handler: adminsearch.PublicSearchHandler{Autocomplete: autocomplete, Recorder: a.searchQueryQueue}})
	if modules["analytics"] {
		routes = append(routes, router.Route{Pattern: "GET /admin/search-insights", Access: router.Protected, Handler: adminsearch.SearchInsightsHandler(a.analyticsStore, requireTenantScope)})
	}
	return routes
}

// pageRoutes registers the pages module; public page views are counted when
// analytics is on and mutations invalidate the llms.txt discovery cache.
func (a *serveApp) pageRoutes() []router.Route {
	modules, pageLLMSCache := a.modules, a.pageLLMSCache
	contentViewQueue, noCookieQueue, noCookieHasher := a.contentViewQueue, a.noCookieQueue, a.noCookieHasher
	if !modules["pages"] {
		return nil
	}
	pageRoutes := pageRouterRoutes(a.pageHTTP.Routes())
	for i := range pageRoutes {
		if strings.HasPrefix(pageRoutes[i].Pattern, "GET /public/pages/") {
			if modules["analytics"] {
				pageRoutes[i].Handler = analytics.NoCookieMiddleware(analytics.ContentViewMiddleware(pageRoutes[i].Handler, contentViewQueue, "page"), noCookieQueue, noCookieHasher)
			}
		} else if !strings.HasPrefix(pageRoutes[i].Pattern, "GET ") {
			original := pageRoutes[i].Handler
			pageRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if pageLLMSCache != nil {
					defer pageLLMSCache.Invalidate()
				}
				original.ServeHTTP(w, r)
			})
		}
	}
	return pageRoutes
}

// homeRoutes registers the locale landing pages, the site root and /docs.
func (a *serveApp) homeRoutes() []router.Route {
	cfg := a.cfg
	localeEnabled := cfg.I18N.Enabled && len(cfg.I18N.Locales) > 0
	return []router.Route{{Pattern: "GET /{locale}/", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := configuredLocale(r.PathValue("locale"), cfg.I18N.Locales)
		if !localeEnabled || locale == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, technicalHomeWithLocales(locale, cfg.I18N.Locales, publicHomePath(locale), demoFrontendEnabled()))
	})}, {Pattern: "GET /{$}", Access: router.Public, Handler: http.HandlerFunc(a.siteRootPage)}, {Pattern: "GET /docs", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")
		if lang != "tr" {
			lang = "en"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Language", lang)
		_, _ = io.WriteString(w, docsPortalHTML(lang))
	})}}
}

// siteRootPage serves "/": locale negotiation, then the configured public
// home page, the news index, or the technical home page.
func (a *serveApp) siteRootPage(w http.ResponseWriter, r *http.Request) {
	cfg, db, modules, siteStore := a.cfg, a.db, a.modules, a.siteStore
	localeEnabled := cfg.I18N.Enabled && len(cfg.I18N.Locales) > 0
	if r.URL.Path == "/" {
		if localeEnabled {
			selected, manual := requestedLocale(r, cfg.I18N.Locales, cfg.I18N.DefaultLocale)
			if manual {
				http.SetCookie(w, &http.Cookie{Name: "hydro_locale", Value: selected, Path: "/", MaxAge: 31536000, HttpOnly: true, Secure: cfg.Security.SecureCookies, SameSite: http.SameSiteLaxMode})
				localeRedirect(w, r, selected)
				return
			}
			if selected != "" && !strings.EqualFold(selected, cfg.I18N.DefaultLocale) {
				localeRedirect(w, r, selected)
				return
			}
		}
		homePath := ""
		if modules["site"] {
			if settings, err := siteStore.ListPublicSettings(r.Context(), db); err == nil {
				homePath = site.ResolvePublicHomePath(settings, "")
			}
		}
		if homePath == "" && modules["news"] {
			homePath = "/public/news/" + url.PathEscape(cfg.I18N.DefaultLocale) + "/html"
		}
		if homePath != "" && homePath != "/" {
			http.Redirect(w, r, homePath, http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, technicalHomeWithLocales(cfg.I18N.DefaultLocale, cfg.I18N.Locales, publicHomePath(cfg.I18N.DefaultLocale), demoFrontendEnabled()))
		return
	}
	http.NotFound(w, r)
}

func (a *serveApp) analyticsRoutes() []router.Route {
	if !a.modules["analytics"] {
		return nil
	}
	cfg, analyticsStore := a.cfg, a.analyticsStore
	return []router.Route{
		{Pattern: "POST /analytics/consent", Access: router.Public, Handler: analytics.ConsentHandler(analyticsStore, cfg.Security.SecureCookies)},
		{Pattern: "POST /analytics/contact", Access: router.Public, Handler: analytics.ContactClickHandler(analyticsStore)},
		{Pattern: "POST /api/v1/feedback", Access: router.Public, Handler: analytics.FeedbackHandler{Store: analyticsStore, Limiter: a.newRateLimiter(8, time.Hour), RequireTenantScope: a.requireTenantScope}},
	}
}

func (a *serveApp) mediaRoutes() []router.Route {
	if !a.modules["media"] {
		return nil
	}
	routes := mediaRouterRoutes(a.mediaHTTP.AdminRoutes())
	return append(routes, mediaRouterRoutes(a.mediaHTTP.Routes())...)
}

// siteRoutes registers the site module; mutations purge cached public HTML.
func (a *serveApp) siteRoutes() []router.Route {
	publicHTMLCache, staticCacheDir := a.publicHTMLCache, a.staticCacheDir
	if !a.modules["site"] {
		return nil
	}
	siteRoutes := siteRouterRoutes(a.siteHTTP.Routes())
	for i := range siteRoutes {
		if !strings.HasPrefix(siteRoutes[i].Pattern, "GET ") {
			original := siteRoutes[i].Handler
			siteRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				original.ServeHTTP(w, r)
				if publicHTMLCache != nil {
					publicHTMLCache.PurgeTag("pages")
					publicHTMLCache.PurgeTag("news")
				}
				if staticCacheDir != "" {
					_ = server.PurgeStaticDiskCache(staticCacheDir)
				}
			})
		}
	}
	return siteRoutes
}

// newsRoutes registers the news module; public article views are counted
// (except previews) and mutations invalidate discovery and HTML caches.
func (a *serveApp) newsRoutes() []router.Route {
	modules, pageLLMSCache := a.modules, a.pageLLMSCache
	publicHTMLCache, staticCacheDir := a.publicHTMLCache, a.staticCacheDir
	contentViewQueue, noCookieQueue, noCookieHasher := a.contentViewQueue, a.noCookieQueue, a.noCookieHasher
	if !modules["news"] {
		return nil
	}
	newsRoutes := newsRouterRoutes(a.newsHTTP.Routes())
	for i := range newsRoutes {
		if strings.HasPrefix(newsRoutes[i].Pattern, "GET /public/news/") {
			if modules["analytics"] {
				original := newsRoutes[i].Handler
				tracked := analytics.NoCookieMiddleware(analytics.ContentViewMiddleware(original, contentViewQueue, "news"), noCookieQueue, noCookieHasher)
				newsRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("preview_token") != "" {
						original.ServeHTTP(w, r)
						return
					}
					tracked.ServeHTTP(w, r)
				})
			}
		} else if !strings.HasPrefix(newsRoutes[i].Pattern, "GET ") {
			original := newsRoutes[i].Handler
			newsRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if pageLLMSCache != nil {
					defer pageLLMSCache.Invalidate()
				}
				if staticCacheDir != "" {
					defer server.PurgeStaticDiskCache(staticCacheDir)
				}
				if publicHTMLCache != nil {
					defer publicHTMLCache.PurgeTag("news")
				}
				original.ServeHTTP(w, r)
			})
		}
	}
	return newsRoutes
}
