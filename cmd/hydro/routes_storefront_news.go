package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/ardaozyildirim/hydrogo/internal/blueprints"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/storefront"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func (a *serveApp) loadNewsData(ctx context.Context, gov tenant.Governance, settings map[string]any, r *http.Request) storefront.NewsData {
	data := storefront.NewsData{
		BrandName:         gov.DisplayName,
		Tagline:           "Bağımsız Dijital Gazete, Derinlemesine Analiz & Teknoloji",
		PrimaryColor:      "#DC2626",
		BGTone:            "#09090B",
		FontPreset:        "editorial_playfair",
		BreakingNews:      "Yapay Zeka Regülasyonlarında Yeni Dönem: Küresel Açık Kaynak Standartları Kabul Edildi",
		ContactEmail:      fmt.Sprintf("editor@%s.%s", gov.TenantID, a.getBaseDomain()),
		ActivePage:        "home",
		WeatherCity:       "İstanbul",
		WeatherTemp:       "18°C",
		WeatherDesc:       "Güneşli",
		PrayerImsak:       "05:42",
		PrayerAksam:       "18:04",
		WhatsAppTipNumber: "905550000000",
		EnableMemberships: true,
		EnableComments:    true,
	}

	if s, ok := settings["brand_name"].(string); ok && s != "" {
		data.BrandName = s
	}
	if s, ok := settings["tagline"].(string); ok && s != "" {
		data.Tagline = s
	}
	if s, ok := settings["primary_color"].(string); ok && s != "" {
		data.PrimaryColor = s
	}
	if s, ok := settings["bg_tone"].(string); ok && s != "" {
		data.BGTone = s
	}
	if s, ok := settings["font_preset"].(string); ok && s != "" {
		data.FontPreset = s
	}
	if s, ok := settings["breaking_news"].(string); ok && s != "" {
		data.BreakingNews = s
	}
	if s, ok := settings["contact_email"].(string); ok && s != "" {
		data.ContactEmail = s
	}
	if s, ok := settings["weather_city"].(string); ok && s != "" {
		data.WeatherCity = s
	}
	if s, ok := settings["weather_temp"].(string); ok && s != "" {
		data.WeatherTemp = s
	}
	if s, ok := settings["weather_desc"].(string); ok && s != "" {
		data.WeatherDesc = s
	}
	if s, ok := settings["prayer_imsak"].(string); ok && s != "" {
		data.PrayerImsak = s
	}
	if s, ok := settings["prayer_aksam"].(string); ok && s != "" {
		data.PrayerAksam = s
	}
	if s, ok := settings["whatsapp_tip_number"].(string); ok && s != "" {
		data.WhatsAppTipNumber = s
	}

	var allArticles []storefront.NewsArticle
	if a.collectionService.Repository != nil {
		if recs, err := a.collectionService.ListRecords(ctx, "news_categories", 20, 0); err == nil {
			for _, rec := range recs {
				data.Categories = append(data.Categories, storefront.NewsCategory{
					Name: strVal(rec.Data["name"]),
					Slug: strVal(rec.Data["slug"]),
				})
			}
		}
		if recs, err := a.collectionService.ListRecords(ctx, "news_articles", 50, 0); err == nil {
			for _, rec := range recs {
				allArticles = append(allArticles, storefront.NewsArticle{
					Title:        strVal(rec.Data["title"]),
					Slug:         strVal(rec.Data["slug"]),
					CategorySlug: strVal(rec.Data["category_slug"]),
					Excerpt:      strVal(rec.Data["excerpt"]),
					Body:         strVal(rec.Data["body"]),
					ImageURL:     strVal(rec.Data["image_url"]),
					AuthorName:   strVal(rec.Data["author_name"]),
					ReadingTime:  intVal(rec.Data["reading_time"]),
					PublishedAt:  strVal(rec.Data["published_at"]),
					IsLead:       boolVal(rec.Data["is_lead"]),
					IsTrending:   boolVal(rec.Data["is_trending"]),
				})
			}
		}
	}

	if len(allArticles) == 0 {
		_, defaultCats, defaultArts := blueprints.GenerateNewsCatalog(data.BrandName, "", "")
		for _, c := range defaultCats {
			data.Categories = append(data.Categories, storefront.NewsCategory{
				Name: strVal(c["name"]),
				Slug: strVal(c["slug"]),
			})
		}
		for _, art := range defaultArts {
			allArticles = append(allArticles, storefront.NewsArticle{
				Title:        strVal(art["title"]),
				Slug:         strVal(art["slug"]),
				CategorySlug: strVal(art["category_slug"]),
				Excerpt:      strVal(art["excerpt"]),
				Body:         strVal(art["body"]),
				ImageURL:     strVal(art["image_url"]),
				AuthorName:   strVal(art["author_name"]),
				ReadingTime:  intVal(art["reading_time"]),
				PublishedAt:  strVal(art["published_at"]),
				IsLead:       boolVal(art["is_lead"]),
				IsTrending:   boolVal(art["is_trending"]),
			})
		}
	}

	p := strings.Trim(r.URL.Path, "/")
	if strings.HasPrefix(p, "haber/") || strings.HasPrefix(p, "news/") {
		slug := strings.TrimPrefix(strings.TrimPrefix(p, "haber/"), "news/")
		for idx, a := range allArticles {
			if strings.EqualFold(a.Slug, slug) {
				artCopy := a
				data.ActiveArticle = &artCopy
				data.ActivePage = "article"
				if idx+1 < len(allArticles) {
					nextCopy := allArticles[idx+1]
					data.NextArticle = &nextCopy
				} else if len(allArticles) > 1 {
					nextCopy := allArticles[0]
					data.NextArticle = &nextCopy
				}
				break
			}
		}
	} else if strings.HasPrefix(p, "kategori/") || strings.HasPrefix(p, "category/") {
		catSlug := strings.TrimPrefix(strings.TrimPrefix(p, "kategori/"), "category/")
		data.ActiveCategory = catSlug
		data.ActivePage = "category"
	} else if strings.Contains(p, "/") {
		parts := strings.Split(p, "/")
		if len(parts) == 2 {
			for _, cat := range data.Categories {
				if strings.EqualFold(cat.Slug, parts[0]) {
					for idx, a := range allArticles {
						if strings.EqualFold(a.Slug, parts[1]) {
							artCopy := a
							data.ActiveArticle = &artCopy
							data.ActivePage = "article"
							if idx+1 < len(allArticles) {
								nextCopy := allArticles[idx+1]
								data.NextArticle = &nextCopy
							} else if len(allArticles) > 1 {
								nextCopy := allArticles[0]
								data.NextArticle = &nextCopy
							}
							break
						}
					}
					break
				}
			}
		}
	} else if p == "yazarlar" {
		data.ActivePage = "columnists"
	} else if p == "galeri" {
		data.ActivePage = "gallery"
	} else if p == "kunye" || p == "imprint" {
		data.ActivePage = "kunye"
	} else if p == "yayin-ilkeleri" {
		data.ActivePage = "yayin-ilkeleri"
	} else if p == "gizlilik-politikasi" || p == "gizlilik" || p == "privacy" {
		data.ActivePage = "gizlilik"
	} else if p == "cerez-politikasi" || p == "cerezler" || p == "cookies" {
		data.ActivePage = "cerez"
	} else if p == "kvkk" || p == "kvkk-aydinlatma-metni" {
		data.ActivePage = "kvkk"
	} else if p != "" && p != "home" && p != "anasayfa" {
		for _, cat := range data.Categories {
			if strings.EqualFold(cat.Slug, p) {
				data.ActiveCategory = cat.Slug
				data.ActivePage = "category"
				break
			}
		}
	}

	data.CurrentDateFormatted = "8 Kasım 2026, Pazar"

	data.MarketTickers = []storefront.MarketTicker{
		{Symbol: "BIST 100", Price: "9.245,30", Change: "+%1.42", IsUp: true},
		{Symbol: "USD/TRY", Price: "34,28", Change: "+%0.08", IsUp: true},
		{Symbol: "EUR/TRY", Price: "37,15", Change: "-%0.12", IsUp: false},
		{Symbol: "Gram Altın", Price: "3.020 TL", Change: "+%0.65", IsUp: true},
		{Symbol: "BTC/USD", Price: "76.420 $", Change: "+%3.18", IsUp: true},
	}
	data.TrendingTags = []string{"YapayZeka", "FinansMimarisi", "AkdenizMinimalizmi", "KuantumCip", "KatiHalPili", "JamesWebb", "Uzay"}

	// Load Columnists from DB collection if exists, else fallback
	if a.collectionService.Repository != nil {
		if recs, err := a.collectionService.ListRecords(ctx, "columnists", 20, 0); err == nil && len(recs) > 0 {
			var colList []storefront.NewsColumnist
			for _, rec := range recs {
				name := strVal(rec.Data["name"])
				initials := ""
				for _, p := range strings.Fields(name) {
					if r, _ := utf8.DecodeRuneInString(p); r != utf8.RuneError {
						initials += string(r)
					}
				}
				if len(initials) > 2 {
					initials = initials[:2]
				}
				slug := strVal(rec.Data["latest_article_slug"])
				if slug == "" {
					slug = strVal(rec.Data["slug"])
				}
				colList = append(colList, storefront.NewsColumnist{
					Name:               name,
					Title:              strVal(rec.Data["title"]),
					Initials:           strings.ToUpper(initials),
					LatestArticleTitle: strVal(rec.Data["latest_article_title"]),
					LatestArticleSlug:  slug,
				})
			}
			if len(colList) > 0 {
				data.Columnists = colList
			}
		}
	}
	if len(data.Columnists) == 0 {
		data.Columnists = []storefront.NewsColumnist{
			{
				Name:               "Prof. Dr. Melih Arslan",
				Title:              "Kıdemli Başekonomist",
				Initials:           "MA",
				LatestArticleTitle: "Küresel Faiz İndirim Döngüsü ve Gelişen Piyasalara Sermaye Akışının Yönü",
				LatestArticleSlug:  "kuresel-faiz-indirim-dongusu",
			},
			{
				Name:               "Selin Doğan",
				Title:              "Teknoloji & İnovasyon Masası",
				Initials:           "SD",
				LatestArticleTitle: "Yapay Zeka ve Yazılım Ekiplerinin Geleceği: Kod Yazarları mı, Sistem Mimarları mı?",
				LatestArticleSlug:  "otonom-yazilim-ajanlari-cagi",
			},
			{
				Name:               "Barış Yılmaz",
				Title:              "Küresel Güvenlik & Dış Politika",
				Initials:           "BY",
				LatestArticleTitle: "Kritik Hammaddeler Savaşı: Nadir Toprak Elementleri ve Tedarik Zinciri",
				LatestArticleSlug:  "kritik-hammadde-savasi",
			},
			{
				Name:               "Zeynep Erdem",
				Title:              "Kültür, Mimari & Kent Masası",
				Initials:           "ZE",
				LatestArticleTitle: "Dijital Göçebelik ve Metropollerin Mekansal Dönüşüm Dinamikleri",
				LatestArticleSlug:  "dijital-gocebelik-ve-metropoller",
			},
		}
	}

	// Load Poll from DB collection if exists, else fallback
	if a.collectionService.Repository != nil {
		if recs, err := a.collectionService.ListRecords(ctx, "polls", 5, 0); err == nil && len(recs) > 0 {
			var activeRec *collections.Record
			for idx := range recs {
				if boolVal(recs[idx].Data["is_active"]) {
					activeRec = &recs[idx]
					break
				}
			}
			if activeRec == nil {
				activeRec = &recs[0]
			}
			rawOpts := strVal(activeRec.Data["options"])
			tot := intVal(activeRec.Data["total_votes"])
			if tot <= 0 {
				tot = 1420
			}
			var pOpts []storefront.NewsPollOption
			for idx, opt := range strings.Split(rawOpts, ",") {
				opt = strings.TrimSpace(opt)
				if opt == "" {
					continue
				}
				pct := 33
				if idx == 0 {
					pct = 64
				} else if idx == 1 {
					pct = 28
				} else {
					pct = 8
				}
				pOpts = append(pOpts, storefront.NewsPollOption{
					Text:       opt,
					Percentage: pct,
					Votes:      int(float64(tot) * float64(pct) / 100.0),
				})
			}
			if len(pOpts) > 0 {
				data.Poll = &storefront.NewsPoll{
					ID:         activeRec.ID,
					Question:   strVal(activeRec.Data["question"]),
					TotalVotes: tot,
					Options:    pOpts,
				}
			}
		}
	}
	if data.Poll == nil {
		data.Poll = &storefront.NewsPoll{
			ID:         "poll-2026-ai",
			Question:   "Yapay zeka modellerinin telifli verilerle eğitilmesi küresel bir lisanslama zorunluluğuna bağlanmalı mı?",
			TotalVotes: 1420,
			Options: []storefront.NewsPollOption{
				{Text: "Evet, kesinlikle zorunlu olmalı", Percentage: 64, Votes: 908},
				{Text: "Hayır, açık araştırma serbest kalmalı", Percentage: 28, Votes: 398},
				{Text: "Kararsızım / Ayrıntılara bağlı", Percentage: 8, Votes: 114},
			},
		}
	}

	// Load Galleries from DB collection if exists, else fallback
	if a.collectionService.Repository != nil {
		if recs, err := a.collectionService.ListRecords(ctx, "galleries", 10, 0); err == nil && len(recs) > 0 {
			var gallList []storefront.NewsGalleryItem
			for _, rec := range recs {
				img := strVal(rec.Data["cover_image"])
				if img == "" {
					img = strVal(rec.Data["image_url"])
				}
				badge := strVal(rec.Data["badge_text"])
				if badge == "" {
					badge = "Özel Dosya"
				}
				gallList = append(gallList, storefront.NewsGalleryItem{
					Title:     strVal(rec.Data["title"]),
					Category:  strVal(rec.Data["category"]),
					ImageURL:  img,
					Slug:      strVal(rec.Data["slug"]),
					BadgeText: badge,
				})
			}
			if len(gallList) > 0 {
				data.GalleryItems = gallList
			}
		}
	}
	if len(data.GalleryItems) == 0 {
		data.GalleryItems = []storefront.NewsGalleryItem{
			{
				Title:     "Kuantum İşlemcilerin Temiz Oda Üretim Süreci",
				Category:  "Teknoloji Dosyası",
				ImageURL:  "https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=800&q=80",
				Slug:      "otonom-yazilim-ajanlari-cagi",
				BadgeText: "12 Fotoğraf",
			},
			{
				Title:     "James Webb Teleskobu ile Keşfedilen Erken Evren Galaksileri",
				Category:  "Uzay & Bilim",
				ImageURL:  "https://images.unsplash.com/photo-1451187580459-43490279c0fa?auto=format&fit=crop&w=800&q=80",
				Slug:      "otonom-yazilim-ajanlari-cagi",
				BadgeText: "Özel Dosya",
			},
			{
				Title:     "Yeni Nesil Katı Hal Piller ve Batarya Fabrikaları",
				Category:  "Enerji & Mobilite",
				ImageURL:  "https://images.unsplash.com/photo-1466611653911-95081537e5b7?auto=format&fit=crop&w=800&q=80",
				Slug:      "yesil-enerjide-kati-hal-pil-devrimi",
				BadgeText: "Araştırma",
			},
		}
	}

	if data.ActiveArticle != nil {
		data.Comments = []storefront.NewsComment{
			{
				ID: "c1", ArticleSlug: data.ActiveArticle.Slug, AuthorName: "Ahmet Yılmaz",
				CreatedAt: "2 saat önce", Content: "Sistem mimarisindeki bu paradigmatik sıçrama, yazılım ekiplerinin organizasyon yapısını da doğrudan etkileyecek. Çok başarılı bir analiz.",
				AvatarInitials: "AY",
			},
			{
				ID: "c2", ArticleSlug: data.ActiveArticle.Slug, AuthorName: "Dr. Zeynep Kaya",
				CreatedAt: "4 saat önce", Content: "Ajanik döngülerin denetlenmesi ve güvenlik sınırlarının belirlenmesi önümüzdeki dönemin en kritik konusu olacak.",
				AvatarInitials: "ZK",
			},
		}
	}

	for i := range allArticles {
		art := allArticles[i]
		if data.ActiveCategory != "" && !strings.EqualFold(art.CategorySlug, data.ActiveCategory) {
			continue
		}
		if art.IsLead && data.LeadArticle == nil {
			leadCopy := art
			data.LeadArticle = &leadCopy
		} else {
			data.Articles = append(data.Articles, art)
		}
		if art.IsTrending && len(data.Trending) < 5 {
			data.Trending = append(data.Trending, art)
		}
	}
	if data.LeadArticle == nil && len(allArticles) > 0 {
		leadCopy := allArticles[0]
		data.LeadArticle = &leadCopy
	}

	// 1. Featured Manset Carousel (1..16)
	mansetCount := len(allArticles)
	if mansetCount > 16 {
		mansetCount = 16
	}
	if mansetCount > 0 {
		data.FeaturedManset = append(data.FeaturedManset, allArticles[:mansetCount]...)
	}

	// 2. Side Manset Cards (up to 5 items for Günün Başlıkları)
	for _, a := range allArticles {
		if len(data.SideManset) >= 5 {
			break
		}
		data.SideManset = append(data.SideManset, a)
	}

	// 3. Trending / Sürmanşet (4 items for top bar)
	for _, a := range allArticles {
		if len(data.Trending) >= 4 {
			break
		}
		already := false
		for _, t := range data.Trending {
			if t.Slug == a.Slug {
				already = true
				break
			}
		}
		if !already {
			data.Trending = append(data.Trending, a)
		}
	}

	// 3. Breaking Articles for Top Marquee Ticker
	breakCount := len(allArticles)
	if breakCount > 6 {
		breakCount = 6
	}
	if breakCount > 0 {
		data.BreakingArticles = append(data.BreakingArticles, allArticles[:breakCount]...)
	}

	// 4. Popular News (Top 6 ranked for horizontal carousel)
	for _, a := range allArticles {
		if a.IsTrending && len(data.PopularNews) < 6 {
			data.PopularNews = append(data.PopularNews, a)
		}
	}
	for _, a := range allArticles {
		if len(data.PopularNews) >= 6 {
			break
		}
		already := false
		for _, pop := range data.PopularNews {
			if pop.Slug == a.Slug {
				already = true
				break
			}
		}
		if !already {
			data.PopularNews = append(data.PopularNews, a)
		}
	}

	// 5. Category Showcase Sections (1 lead + up to 4 sub cards per category)
	for _, cat := range data.Categories {
		var catArts []storefront.NewsArticle
		for _, a := range allArticles {
			if strings.EqualFold(a.CategorySlug, cat.Slug) {
				catArts = append(catArts, a)
			}
		}
		if len(catArts) >= 2 {
			lead := catArts[0]
			var rest []storefront.NewsArticle
			if len(catArts) > 1 {
				limit := len(catArts)
				if limit > 5 {
					limit = 5
				}
				rest = catArts[1:limit]
			}
			data.CategorySections = append(data.CategorySections, storefront.NewsCategorySection{
				Category:    cat,
				LeadArticle: lead,
				Articles:    rest,
			})
		}
	}

	return data
}
