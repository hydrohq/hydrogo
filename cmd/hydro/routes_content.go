package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	adminsearch "github.com/ardaozyildirim/hydrogo/internal/search"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
	trashstore "github.com/ardaozyildirim/hydrogo/internal/trash/sqlstore"
	trashtransport "github.com/ardaozyildirim/hydrogo/internal/trash/transport"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// collectionRoutes registers the YAML content types under /api/v1/content
// (public reads) and /api/v1/admin/content (admin JSON API) and keeps the
// loaded definitions for the admin schema.
func (a *serveApp) collectionRoutes() ([]router.Route, error) {
	ctx, cfg, db := a.ctx, a.cfg, a.db
	if !a.modules["collections"] {
		return nil, nil
	}
	collectionDB := collectionstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope, MaxRecords: cfg.Limits.RecordsPerType}
	collectionDB.Events = contentEvents(cfg, outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope})
	collectionService := collections.Service{Repository: collectionDB}
	a.collectionService = collectionService
	a.contentLocales = contenttypes.NewLocales(cfg.I18N.Enabled, cfg.I18N.DefaultLocale, cfg.I18N.Locales)
	definitions, contentRoutes, err := contentTypeRoutesWithContext(ctx, cfg.ContentTypesDir, collectionService, a.getTenantScopes(), a.contentLocales, a.editPresence, a.hookCaller(), a.newRateLimiter(120, time.Minute), a.realtimeHub, a.resolveContentPrincipal, a.getContentTypes, a.contentTypesForContext)
	if err != nil {
		return nil, err
	}
	a.stateMu.Lock()
	a.contentTypes = definitions
	a.stateMu.Unlock()
	if err := checkHookServices(cfg.Hooks, definitions); err != nil {
		return nil, err
	}
	if names := contenttypes.PermissionNames(definitions); len(names) > 0 && a.authSvc != nil {
		catalog, err := auth.NewPermissionCatalog(names...)
		if err != nil {
			return nil, fmt.Errorf("content types: permissions: %w", err)
		}
		if err := a.authSvc.SetPermissionCatalog(catalog); err != nil {
			return nil, err
		}
	}
	return contentRoutes, nil
}

func (a *serveApp) resolveContentPrincipal(r *http.Request) contenttypes.Principal {
	if user := auth.UserFromContext(r.Context()); user != nil && user.Active {
		return contenttypes.Principal{
			ID:        user.ID,
			Email:     user.Email,
			Role:      string(user.Role),
			IsAdmin:   user.Role == auth.RoleAdmin,
			IsMember:  false,
			Anonymous: false,
		}
	}
	if a.modules["members"] && a.memberService.Sessions != nil {
		if cookie, err := r.Cookie(members.SessionCookieName); err == nil && cookie.Value != "" {
			if _, member, err := a.memberService.ValidateSession(r.Context(), cookie.Value); err == nil && member != nil && member.Active {
				return contenttypes.Principal{
					ID:        member.ID,
					Email:     member.Email,
					Role:      "member",
					IsAdmin:   false,
					IsMember:  true,
					Anonymous: false,
				}
			}
		}
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token := strings.TrimSpace(authHeader[7:])
			if _, member, err := a.memberService.ValidateSession(r.Context(), token); err == nil && member != nil && member.Active {
				return contenttypes.Principal{
					ID:        member.ID,
					Email:     member.Email,
					Role:      "member",
					IsAdmin:   false,
					IsMember:  true,
					Anonymous: false,
				}
			}
		}
	}
	return contenttypes.Principal{Anonymous: true}
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
		if pageRoutes[i].Pattern == "GET /llms.txt" {
			origLLMs := pageRoutes[i].Handler
			pageRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if a.isStorefrontTenant(r) {
					a.storefrontLLMsHandler(w, r)
					return
				}
				origLLMs.ServeHTTP(w, r)
			})
			continue
		}
		if pageRoutes[i].Pattern == "GET /api/v1/pages/{locale}/{slug}" {
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
	return mediaRouterRoutes(a.mediaHTTP.Routes())
}

// siteRoutes registers the site module; mutations purge cached public HTML.
func (a *serveApp) siteRoutes() []router.Route {
	publicHTMLCache, staticCacheDir := a.publicHTMLCache, a.staticCacheDir
	if !a.modules["site"] {
		return nil
	}
	siteRoutes := siteRouterRoutes(a.siteHTTP.Routes())
	for i := range siteRoutes {
		if siteRoutes[i].Pattern == "GET /sitemap.xml" {
			origSitemap := siteRoutes[i].Handler
			siteRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if a.isStorefrontTenant(r) {
					a.storefrontSitemapHandler(w, r)
					return
				}
				origSitemap.ServeHTTP(w, r)
			})
			continue
		}
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
		if newsRoutes[i].Pattern == "GET /api/v1/news/{locale}/{slug}" {
			if modules["analytics"] {
				newsRoutes[i].Handler = analytics.NoCookieMiddleware(analytics.ContentViewMiddleware(newsRoutes[i].Handler, contentViewQueue, "news"), noCookieQueue, noCookieHasher)
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
