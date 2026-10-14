package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

// drawDemoCoverImage paints an abstract, editorial-looking cover placeholder
// for seed media: a soft gradient field tinted from (r,g,b), not a "tech
// dashboard" diagram. The id selects one of a few gradient compositions
// (hashed, so it's stable across runs) so the handful of seed images reused
// across many demo articles don't all read as copies of the same graphic.
func drawDemoCoverImage(img *image.RGBA, id string, width, height int, r, g, b uint8) {
	hash := fnv32(id)
	variant := hash % 4
	// A deep, desaturated shade of the same hue anchors the gradient so the
	// result reads as a toned photo rather than a flat color swatch.
	deepR, deepG, deepB := uint8(float64(r)*0.22), uint8(float64(g)*0.22), uint8(float64(b)*0.22+6)

	lerp := func(a, c uint8, t float64) uint8 {
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		return uint8(float64(a) + (float64(c)-float64(a))*t)
	}

	switch variant {
	case 0: // diagonal: light corner to deep corner
		diag := math.Hypot(float64(width), float64(height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				t := (float64(x) + float64(y)) / diag
				img.Set(x, y, color.RGBA{R: lerp(deepR, r, 1-t), G: lerp(deepG, g, 1-t), B: lerp(deepB, b, 1-t), A: 255})
			}
		}
	case 1: // radial glow offset toward one corner (varies by hash)
		cx := float64(width) * (0.3 + 0.4*float64(hash%7)/6)
		cy := float64(height) * (0.3 + 0.4*float64((hash/7)%7)/6)
		maxDist := math.Hypot(float64(width), float64(height)) * 0.65
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				t := math.Hypot(float64(x)-cx, float64(y)-cy) / maxDist
				img.Set(x, y, color.RGBA{R: lerp(r, deepR, t), G: lerp(g, deepG, t), B: lerp(b, deepB, t), A: 255})
			}
		}
	case 2: // horizontal duotone with a soft blended seam
		seam := float64(height) * (0.35 + 0.3*float64(hash%5)/4)
		band := float64(height) * 0.25
		for y := 0; y < height; y++ {
			t := (float64(y) - (seam - band)) / (2 * band)
			for x := 0; x < width; x++ {
				img.Set(x, y, color.RGBA{R: lerp(r, deepR, t), G: lerp(g, deepG, t), B: lerp(b, deepB, t), A: 255})
			}
		}
	default: // vertical gradient with a thin accent horizon line
		horizon := height * (55 + int(hash%20)) / 100
		for y := 0; y < height; y++ {
			t := float64(y) / float64(height)
			for x := 0; x < width; x++ {
				img.Set(x, y, color.RGBA{R: lerp(r, deepR, t), G: lerp(g, deepG, t), B: lerp(b, deepB, t), A: 255})
			}
		}
		for x := 0; x < width; x++ {
			img.Set(x, horizon, color.RGBA{R: minUint8(r+50, 255), G: minUint8(g+50, 255), B: minUint8(b+50, 255), A: 160})
		}
	}

	// Gentle vignette so edges aren't flat, without any diagram motifs.
	cx, cy := float64(width)/2, float64(height)/2
	maxDist := math.Hypot(cx, cy)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dist := math.Hypot(float64(x)-cx, float64(y)-cy) / maxDist
			if dist > 0.7 {
				orig := img.RGBAAt(x, y)
				darken := (dist - 0.7) / 0.3 * 0.35
				img.Set(x, y, color.RGBA{
					R: uint8(float64(orig.R) * (1 - darken)),
					G: uint8(float64(orig.G) * (1 - darken)),
					B: uint8(float64(orig.B) * (1 - darken)),
					A: 255,
				})
			}
		}
	}
}

func fnv32(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func minUint8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

func seedDemo(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("seed-demo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	confirmDemo := flags.Bool("confirm-demo", false, "Confirm that this database is a local demo target")
	forceProd := flags.Bool("force-prod", false, "Allow demo writes when the deployment environment is production")
	count := flags.Int("count", 20, "Number of demo news records to create (1-200)")
	locale := flags.String("locale", "", "Demo content locale: tr or en (defaults to configured locale)")
	clean := flags.Bool("clean", false, "Remove HydroGo demo records before seeding")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*confirmDemo {
		return errors.New("seed requires --confirm-demo; use only on a disposable local database")
	}
	if *count < 1 || *count > 200 {
		return errors.New("--count must be between 1 and 200")
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	demoLocale := strings.TrimSpace(cfg.I18N.DefaultLocale)
	if demoLocale == "" {
		demoLocale = "tr"
	}
	if *locale != "" {
		demoLocale = strings.ToLower(strings.TrimSpace(*locale))
	}
	if demoLocale != "tr" && demoLocale != "en" {
		return errors.New("--locale must be tr or en")
	}
	if isProductionEnvironment() && !*forceProd {
		return errors.New("seed is disabled in production; pass --force-prod only for an intentional demo deployment")
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()
	if *clean {
		if err := cleanDemoData(ctx, db, cfg.Database.Dialect, cfg.Media.Dir); err != nil {
			return fmt.Errorf("clean demo data: %w", err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Demo member data is deliberately isolated from admin users. It gives the
	// public member portal a real, repeatable account flow without exposing any
	// administrator capability.
	demoMemberHash, err := auth.HashPassword("memberpass123", auth.DefaultArgon2Params)
	if err != nil {
		return fmt.Errorf("seed demo member password: %w", err)
	}
	if cfg.Database.Dialect == database.MySQL {
		if err := seedExecMySQL(ctx, db,
			`INSERT IGNORE INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at) VALUES (?,?,?,1,?,?,?,?)`,
			"demo-member-portal", "member@hydrogo.local", demoMemberHash, now, now, now, now); err != nil {
			return fmt.Errorf("seed demo member: %w", err)
		}
	} else if err := seedExec(ctx, db, cfg.Database.Dialect,
		`INSERT INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at) VALUES ($1,$2,$3,TRUE,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING`,
		`INSERT OR IGNORE INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at) VALUES (?,?,?,1,?,?,?,?)`,
		"demo-member-portal", "member@hydrogo.local", demoMemberHash, now, now, now, now); err != nil {
		return fmt.Errorf("seed demo member: %w", err)
	}

	if err := os.MkdirAll(cfg.Media.Dir, 0o750); err != nil {
		return fmt.Errorf("seed media directory: %w", err)
	}

	seedMedia := func(id, filename string, width, height int, r, g, b uint8) error {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		drawDemoCoverImage(img, id, width, height, r, g, b)

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("encode seed image %s: %w", id, err)
		}
		payload := buf.Bytes()
		hasher := sha256.New()
		hasher.Write(payload)
		storageKey := hex.EncodeToString(hasher.Sum(nil)) + ".png"
		filePath := filepath.Join(cfg.Media.Dir, storageKey)
		if err := os.WriteFile(filePath, payload, 0o600); err != nil {
			return fmt.Errorf("write seed media file %s: %w", filePath, err)
		}
		sizeBytes := int64(len(payload))
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `INSERT IGNORE INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES (?,?,?,'image/png',?,?,?,'public','ready',?,?)`, id, storageKey, filename, sizeBytes, width, height, now, now); err != nil {
				return fmt.Errorf("seed media asset %s: %w", id, err)
			}
			if err := seedExecMySQL(ctx, db, `UPDATE media_assets SET storage_key=?,original_filename=?,size_bytes=?,width=?,height=?,visibility='public',status='ready',updated_at=? WHERE id=?`, storageKey, filename, sizeBytes, width, height, now, id); err != nil {
				return err
			}
		} else {
			if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES ($1,$2,$3,'image/png',$4,$5,$6,'public','ready',$7,$7) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES (?1,?2,?3,'image/png',?4,?5,?6,'public','ready',?7,?7)`, id, storageKey, filename, sizeBytes, width, height, now); err != nil {
				return fmt.Errorf("seed media asset %s: %w", id, err)
			}
			if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE media_assets SET storage_key=$1,original_filename=$2,size_bytes=$3,width=$4,height=$5,visibility='public',status='ready',updated_at=$6 WHERE id=$7`, `UPDATE media_assets SET storage_key=?1,original_filename=?2,size_bytes=?3,width=?4,height=?5,visibility='public',status='ready',updated_at=?6 WHERE id=?7`, storageKey, filename, sizeBytes, width, height, now, id); err != nil {
				return err
			}
		}
		var storedKey string
		query := `SELECT storage_key FROM media_assets WHERE id=$1`
		if cfg.Database.Dialect == database.SQLite {
			query = `SELECT storage_key FROM media_assets WHERE id=?1`
		} else if cfg.Database.Dialect == database.MySQL {
			query = `SELECT storage_key FROM media_assets WHERE id=?`
		}
		if err := db.QueryRowContext(ctx, query, id).Scan(&storedKey); err != nil {
			return fmt.Errorf("seed media asset %s was not persisted: %w", id, err)
		}
		if storedKey != storageKey {
			return fmt.Errorf("seed media asset %s storage key collision: got %q, want %q", id, storedKey, storageKey)
		}
		return nil
	}

	mediaFixtures := []struct {
		id, filename string
		w, h         int
		r, g, b      uint8
	}{
		{"demo-media-hero-home", "hero-home.png", 800, 450, 21, 94, 239},
		{"demo-media-hero-showcase", "hero-showcase.png", 800, 450, 15, 23, 42},
		{"demo-media-news-1", "news-2-0.png", 800, 450, 37, 99, 235},
		{"demo-media-news-2", "news-guide.png", 800, 450, 14, 116, 144},
		// Keep the generated payload distinct from hero-showcase; storage keys
		// are content hashes and duplicate bytes would be ignored by the
		// unique-key constraint, silently dropping this fixture.
		{"demo-media-news-3", "news-security.png", 800, 450, 16, 24, 43},
		{"demo-media-news-4", "news-db.png", 800, 450, 79, 70, 229},
		{"demo-media-news-5", "news-jobs.png", 800, 450, 5, 150, 105},
		{"demo-media-news-6", "news-cache.png", 800, 450, 217, 119, 6},
		{"demo-media-news-7", "news-theme.png", 800, 450, 192, 38, 211},
		{"demo-media-news-8", "news-commerce.png", 800, 450, 225, 29, 72},
		{"demo-media-prod-1", "hydrogo-starter.png", 400, 300, 30, 41, 59},
		{"demo-media-prod-2", "hydrogo-growth.png", 400, 300, 37, 99, 235},
		{"demo-media-prod-3", "hydrogo-enterprise.png", 400, 300, 15, 23, 42},
		{"demo-media-prod-4", "hydro-bottle.png", 400, 300, 2, 132, 199},
		{"demo-media-prod-5", "hydro-shirt.png", 400, 300, 71, 85, 105},
		{"demo-media-prod-6", "hydro-keyboard.png", 400, 300, 51, 65, 85},
		{"demo-media-prod-7", "hydro-backpack.png", 400, 300, 79, 70, 229},
		{"demo-media-prod-8", "hydro-edge-hub.png", 400, 300, 16, 185, 129},
	}
	for _, mf := range mediaFixtures {
		if err := seedMedia(mf.id, mf.filename, mf.w, mf.h, mf.r, mf.g, mf.b); err != nil {
			return fmt.Errorf("seed media %s: %w", mf.id, err)
		}
	}

	seedPage := func(id, slug, title, body, blocks string) error {
		if demoLocale == "en" {
			switch id {
			case "demo-home":
				title, body = "HydroGo Platform", "Manage your site pages, articles, and media from one workspace."
				blocks = `[{"type":"hero","payload":{"title":"Manage your site in one place","subtitle":"A focused workspace for pages, news, and media.","media_asset_id":"demo-media-hero-home","button_text":"Explore the examples","button_url":"/public/pages/en/showcase/html"}},{"type":"faq","payload":{"items":[{"question":"What can I manage?","answer":"Manage pages, news, and media. Additional features depend on enabled modules."},{"question":"Is this production content?","answer":"No. This page contains demo content for evaluation."}]}}]`
			case "demo-showcase":
				title, body = "HydroGo Showcase", "Examples of pages and reusable content blocks."
				blocks = `[{"type":"hero","payload":{"title":"HydroGo component examples","subtitle":"A sample gallery of reusable page blocks.","media_asset_id":"demo-media-hero-showcase","button_text":"Browse articles","button_url":"/public/news/en/html"}},{"type":"faq","payload":{"items":[{"question":"What does this page show?","answer":"It shows how reusable page blocks appear in a published theme."},{"question":"Are these commercial terms?","answer":"No. Demo content does not define prices or service commitments."}]}}]`
			case "demo-about":
				title, body = "About HydroGo", "A configurable application core for modular content and commerce projects."
				blocks = `[{"type":"hero","payload":{"title":"HydroGo architecture","subtitle":"A modular foundation for content and commerce.","media_asset_id":"demo-media-hero-about","button_text":"Read the updates","button_url":"/public/news/en/html"}},{"type":"faq","payload":{"items":[{"question":"Can modules be enabled separately?","answer":"Editions choose the modules they need."},{"question":"Is this a commercial offer?","answer":"No. Demo content is illustrative only."}]}}]`
			}
		}
		localePath := url.PathEscape(demoLocale)
		blocks = strings.ReplaceAll(strings.ReplaceAll(blocks, "/public/pages/tr/", "/public/pages/"+localePath+"/"), "/public/news/tr/", "/public/news/"+localePath+"/")
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `INSERT IGNORE INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES (?,?,1,?,?,?,'published',1,1,1,?,?,?,?)`, id, title, demoLocale, slug, body, blocks, blocks, now, now); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES ($1,$2,1,$3,$4,$5,'published',1,1,1,$6,$6,$7,$7) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES (?1,?2,1,?3,?4,?5,'published',1,1,1,?6,?6,?7,?7)`, id, title, demoLocale, slug, body, blocks, now); err != nil {
			return err
		}
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `UPDATE pages SET locale=?,slug=?,title=?,body=?,blocks=?,published_locale=?,published_slug=?,published_title=?,published_body=?,published_blocks=? WHERE id=?`, demoLocale, slug, title, body, blocks, demoLocale, slug, title, body, blocks, id); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE pages SET locale=$1,slug=$2,title=$3,body=$4,blocks=$5,published_locale=$1,published_slug=$2,published_title=$3,published_body=$4,published_blocks=$5 WHERE id=$6`, `UPDATE pages SET locale=?1,slug=?2,title=?3,body=?4,blocks=?5,published_locale=?1,published_slug=?2,published_title=?3,published_body=?4,published_blocks=?5 WHERE id=?6`, demoLocale, slug, title, body, blocks, id); err != nil {
			return err
		}
		if cfg.Database.Dialect == database.MySQL {
			return seedExecMySQL(ctx, db, `INSERT IGNORE INTO page_revisions (id,page_id,revision,locale,slug,title,body,blocks,created_at) VALUES (?,?,1,?,?,?,?,?,?)`, id+"-rev-1", id, demoLocale, slug, title, body, blocks, now)
		}
		return seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,blocks,created_at) VALUES ($1,$2,1,$3,$4,$5,$6,$7,$8) ON CONFLICT (page_id,revision) DO NOTHING`, `INSERT OR IGNORE INTO page_revisions (id,page_id,revision,locale,slug,title,body,blocks,created_at) VALUES (?1,?2,1,?3,?4,?5,?6,?7,?8)`, id+"-rev-1", id, demoLocale, slug, title, body, blocks, now)
	}

	homeBlocks := `[{"type":"hero","payload":{"title":"İçeriğinizi ve mağazanızı tek yerde yönetin","subtitle":"Sayfalar, haberler, medya ve ürün kataloğu için sade bir yönetim deneyimi.","media_asset_id":"demo-media-hero-home","button_text":"Yetenekleri incele →","button_url":"/public/pages/tr/showcase/html"}},{"type":"faq","payload":{"items":[{"question":"Hangi içerikleri yönetebilirim?","answer":"Sayfalar, haberler, medya varlıkları ve Commerce modülü açıksa ürünler, varyantlar ve siparişler yönetilebilir."},{"question":"SQLite ve PostgreSQL destekleniyor mu?","answer":"Her iki veritabanı için aynı davranış sözleşmeleri ve migration akışları bulunur; deployment öncesi ortamınıza uygun doğrulama yapılmalıdır."},{"question":"Üye hesabı nasıl çalışır?","answer":"Üyeler public frontend üzerinden kayıt olabilir, giriş yapabilir, sepet ve kendi sipariş geçmişlerini görebilir. Admin Studio erişimleri ayrıdır."},{"question":"Ödeme canlı mı?","answer":"Demo ortamında ödeme sağlayıcı akışı yalnızca yapılandırılmış sandbox veya test sağlayıcısı ile çalışır; canlı anahtarlar ayrıca kurulmalıdır."}]}},{"type":"cta","payload":{"title":"Kendi içerik akışınızı oluşturun","button_text":"Studio'ya git →","button_url":"/admin"}}]`
	showcaseBlocks := `[{"type":"hero","payload":{"title":"HydroGo bileşen örnekleri","subtitle":"Bu sayfa, kurulumla gelen örnek blokların bir vitrini; ticari teklif, müşteri yorumu veya hizmet taahhüdü değildir.","media_asset_id":"demo-media-hero-showcase","button_text":"Mağaza örneğini incele →","button_url":"/shop"}},{"type":"faq","payload":{"items":[{"question":"Bu sayfa neyi gösterir?","answer":"Hero, CTA ve SSS gibi yayınlanabilir sayfa bloklarının tema tarafından nasıl sunulduğunu gösterir."},{"question":"Fiyatlar veya hizmet koşulları burada tanımlı mı?","answer":"Hayır. Ticari fiyat, sözleşme, SLA ve hukuki koşullar ilgili işletme tarafından ayrıca yayınlanmalıdır."},{"question":"Özel bloklar nasıl eklenir?","answer":"Tema slotları ve sürümlü modül sözleşmeleri üzerinden, proje gereksinimine göre geliştirilir."}]}},{"type":"cta","payload":{"title":"Yayınlanan içerik örneklerini inceleyin","button_text":"Haberlere göz at →","button_url":"/public/news/tr/html"}}]`
	aboutBlocks := `[{"type":"hero","payload":{"title":"HydroGo Vizyonu ve Mimari Felsefesi","subtitle":"Modüler içerik ve ticaret ihtiyaçları için yapılandırılabilir bir uygulama çekirdeği.","media_asset_id":"demo-media-hero-about","button_text":"Haber Merkezini Gör →","button_url":"/public/news/tr/html"}},{"type":"faq","payload":{"items":[{"question":"Modüler CMS + Commerce yaklaşımı nedir?","answer":"HydroCore temel sözleşmeleri tanımlar. Pages, News ve Commerce modülleri gerektiğinde etkinleştirilebilir."},{"question":"Güvenlik nasıl ele alınır?","answer":"Kimlik, oturum, CSRF ve içerik güvenliği kontrolleri yapılandırılan modüllerde uygulanır. Canlıya çıkmadan önce ortam ve entegrasyon doğrulaması yapılmalıdır."},{"question":"Headless ve API desteği var mı?","answer":"Etkin API modülleri REST ve GraphQL uç noktaları sunar; kullanılabilirlik kurulum yapılandırmasına bağlıdır."}]}},{"type":"cta","payload":{"title":"Geliştirici veya İçerik Yöneticisi misiniz?","button_text":"Studio Çalışma Alanı →","button_url":"/admin"}}]`

	if err := seedPage("demo-home", "demo", "HydroGo Platformu", "HydroGo kurumsal içerik yönetim ve e-ticaret platformuna hoş geldiniz.", homeBlocks); err != nil {
		return fmt.Errorf("seed demo home: %w", err)
	}
	if err := seedPage("demo-showcase", "showcase", "HydroGo Showcase", "Tüm blok tiplerini ve mimari yetenekleri keşfedin.", showcaseBlocks); err != nil {
		return fmt.Errorf("seed demo showcase: %w", err)
	}
	if err := seedPage("demo-about", "about", "HydroGo Hakkında", "HydroCore, modüller ve Client Edition yaklaşımı.", aboutBlocks); err != nil {
		return fmt.Errorf("seed demo about: %w", err)
	}

	newsList := []demoNewsSeed{
		{
			id:           "demo-news-release-v2",
			slug:         "hydrogo-2-0-yayinda",
			title:        "HydroGo 2.0 Yayında: Zengin Editör, Gelişmiş Güvenlik ve Modern UI",
			excerpt:      "Yeni sürüm ile birlikte Quill tabanlı zengin metin editörü, key takeaways ve etiket yönetimi, CSP sıkılaştırması ve baştan tasarlanan arayüz kullanıma sunuldu.",
			takeaways:    "- Quill tabanlı zengin metin editörü tüm biçimlendirme araçlarıyla entegre edildi.\n- XSS ve SVG data URL vektörlerine karşı çok katmanlı sanitizasyon getirildi.\n- Kategori, etiket ve öne çıkan içerik rozetleri public haber odasına eklendi.\n- Dev server ve Admin Studio modern mavi tasarım sistemine kavuştu.",
			category:     "Ürün Güncellemeleri",
			author:       "HydroGo Editoryal",
			coverMediaID: "demo-media-news-1",
			featured:     true,
			tags:         "hydrogo, release, security, richtext, go",
			body:         `{"ops":[{"insert":"HydroGo 2.0 İle İçerik Yönetimi Yeni Bir Boyuta Taşınıyor\n","attributes":{"header":1}},{"insert":"HydroGo ekibi olarak, platformun en kapsamlı güncellemesi olan 2.0 sürümünü duyurmaktan büyük heyecan duyuyoruz. Bu sürümde editoryal deneyimi güçlendirirken kurumsal güvenlik gereksinimlerini en üst seviyede karşılamayı hedefledik.\n\n"},{"insert":"Öne Çıkan Başlıca Yenilikler:\n","attributes":{"header":2}},{"insert":"Zengin Metin Editörü: ","attributes":{"bold":true}},{"insert":"Başlık hiyerarşisi, kalın/italik/altı çizili stiller, kod blokları, alıntılar ve medya entegrasyonu.\n"},{"insert":"Editoryal Metadata: ","attributes":{"bold":true}},{"insert":"Haber kategorileri, etiketler, yazar bilgisi, özet ve önemli çıkarımlar (Key Takeaways).\n"},{"insert":"Güvenlik Sertleştirmesi: ","attributes":{"bold":true}},{"insert":"Sıkı Content Security Policy (CSP), CSRF token koruması ve XSS temizleme filtreleri.\n\n"},{"insert":"\"Güvenilir, tip güvenli ve modern bir CMS deneyimi için HydroCore mimarisini her geçen gün daha da ileri taşıyoruz.\"\n","attributes":{"blockquote":true}},{"insert":"Geliştirici deneyimini keşfetmek için Admin Studio'ya giriş yapabilir veya API dokümantasyonunu inceleyebilirsiniz.\n"}]}`,
		},
		{
			id:           "demo-news-guide",
			slug:         "hydrogo-ile-baslangic-rehberi",
			title:        "Geliştiriciler İçin HydroGo Başlangıç ve Entegrasyon Rehberi",
			excerpt:      "Admin Studio üzerinden içerik ve e-ticaret kataloğu yönetimi, REST ve GraphQL API uç noktalarının tüketimi hakkında adım adım kılavuz.",
			takeaways:    "- Admin Studio içerisinden Sayfalar, Haberler ve Commerce ürünleri yönetilebilir.\n- REST API üzerinden tüm yayınlanmış verilere anında JSON formatında erişilir.\n- Headless starter'lar bağımsız frontend mimarileri kurmayı kolaylaştırır.",
			category:     "Rehberler",
			author:       "HydroGo Core Team",
			coverMediaID: "demo-media-news-2",
			featured:     false,
			tags:         "guide, tutorial, api, headless, cms",
			body:         `{"ops":[{"insert":"HydroGo Mimarisine Hızlı Bir Bakış\n","attributes":{"header":1}},{"insert":"HydroGo, modern web uygulamaları ve e-ticaret projeleri için geliştirilmiş yüksek performanslı bir Go CMS çekirdeğidir.\n\n"},{"insert":"1. Adım: Admin Studio ile İçerik Oluşturma\n","attributes":{"header":2}},{"insert":"Admin paneline erişerek yeni sayfalar oluşturabilir, hero, CTA, FAQ gibi blokları sürükleyip yapılandırabilir ve haberlerinizi yayına alabilirsiniz.\n\n"},{"insert":"2. Adım: API Uç Noktalarını Bağlama\n","attributes":{"header":2}},{"insert":"Frontend uygulamanızdan ","attributes":{}},{"insert":"/api/pages","attributes":{"code":true}},{"insert":" veya ","attributes":{}},{"insert":"/api/commerce/products","attributes":{"code":true}},{"insert":" adreslerine istek atarak içerikleri doğrudan tüketebilirsiniz.\n"}]}`,
		},
		{
			id:           "demo-news-security",
			slug:         "kurumsal-seviyede-cms-guvenligi",
			title:        "HydroGo Güvenlik Mimarisi: Çok Katmanlı Koruma ve Sıfır Sızıntı",
			excerpt:      "Argon2id parola şifreleme, TOTP MFA, CSP nonces ve SQL enjeksiyonlarına karşı parameterized query standartları.",
			takeaways:    "- Tüm formlar dinamik CSRF token ve zaman aşımı korumalıdır.\n- Parolalar endüstri standardı Argon2id ile hash'lenir.\n- Çift veritabanı desteği (PostgreSQL & SQLite) kontrat testleri ile korunur.",
			category:     "Güvenlik & Mimari",
			author:       "Güvenlik Mühendisliği",
			coverMediaID: "demo-media-news-3",
			featured:     false,
			tags:         "security, mfa, argon2id, csp, database",
			body:         `{"ops":[{"insert":"Kurumsal Güvenlik Standartlarımız\n","attributes":{"header":1}},{"insert":"HydroGo, finansal ve hassas veri taşıyan uygulamalarda güvenle çalışacak şekilde tasarlanmıştır. Kimlik doğrulama, yetkilendirme ve veri yaşam döngüsü kuralları ADR-0013 ve ADR-0014 standartlarına uyum hedefiyle tasarlanmıştır.\n\n"},{"insert":"Denetim ve İzlenebilirlik:\n","attributes":{"header":2}},{"insert":"Tüm kritik yönetici hareketleri (içerik yayınlama, kullanıcı davetleri, iade işlemleri) değiştirilemez audit log tablosunda kayıt altına alınır.\n"}]}`,
		},
		{
			id:           "demo-news-db-portability",
			slug:         "veritabani-tasinabilirligi-postgresql-sqlite",
			title:        "Veritabanı Taşınabilirliği: SQLite'tan PostgreSQL'e Kesintisiz Geçiş",
			excerpt:      "Dialect bağımsız repository katmanı ve SQL injection riskini azaltan parameterized query standartları.",
			takeaways:    "- Kod tabanında dialect sızıntısına izin verilmez.\n- SQLite yerel geliştirme ve kompakt profil için idealdir.\n- PostgreSQL yüksek eşzamanlılık ve kümeleme desteği sunar.\n- Çift veritabanı davranış testleri her CI adımında doğrulanır.",
			category:     "Veritabanı",
			author:       "Veri Altyapı Ekibi",
			coverMediaID: "demo-media-news-4",
			featured:     false,
			tags:         "database, postgresql, sqlite, migrations, adr0002",
			body:         `{"ops":[{"insert":"Dialect Bağımsız Veri Katmanı Mimarisi\n","attributes":{"header":1}},{"insert":"HydroGo'da tüm SQL sorguları ve migrasyonlar hem SQLite hem de PostgreSQL motorlarında eşit seviyede test edilir. Bu sayede geliştiriciler yerel ortamda SQLite ile sıfır kurulumla başlarken prodüksiyonda PostgreSQL'e güvenle geçebilirler.\n\n"},{"insert":"Sıfır Dialect Sızıntısı Prensibi:\n","attributes":{"header":2}},{"insert":"Domain ve servis katmanları veritabanı sürücülerine doğrudan bağımlı değildir; transactor ve repository arayüzleri üzerinden soyutlanmıştır.\n"}]}`,
		},
		{
			id:           "demo-news-job-engine",
			slug:         "arka-plan-is-yonetimi-ve-outbox-pattern",
			title:        "Arka Plan İş Yönetimi: Güvenilir Olay Dağıtımı ve Dead Letter Kuyrukları",
			excerpt:      "Outbox pattern ile veritabanı işlemlerini atomik olarak olaylara dönüştüren, yeniden deneme ve hata kurtarma yetenekli worker motoru.",
			takeaways:    "- Veritabanı işlemi ve olay kaydı aynı transaction içinde tamamlanır.\n- Başarısız işler üstel geri çekilme ile yeniden denenir.\n- Çözülemeyen işler dead-letter kuyruğuna alınır ve admin tarafından incelenebilir.",
			category:     "Mühendislik",
			author:       "Sistem Mühendisliği",
			coverMediaID: "demo-media-news-5",
			featured:     false,
			tags:         "jobs, outbox, workers, reliability, queues",
			body:         `{"ops":[{"insert":"Güvenilir Asenkron İşleme: Outbox Pattern\n","attributes":{"header":1}},{"insert":"HydroGo'nun arka plan iş motoru, e-posta gönderimi, webhook tetiklemeleri ve arama indeksi güncellemeleri gibi operasyonları ana HTTP iş parçacığından ayırarak web yanıt sürelerini milisaniyeler seviyesinde tutar.\n"}]}`,
		},
		{
			id:           "demo-news-cache-strategy",
			slug:         "onbellekleme-stratejileri-ve-damgalanma-korumasi",
			title:        "Önbellekleme Stratejileri: Stampede Koruması ve Çok Katmanlı Geçersiz Kılma",
			excerpt:      "Yüksek trafikli sayfa ve katalog sorgularında bellek içi önbellekleme, singleflight stampede koruması ve otomatik invalidation.",
			takeaways:    "- Singleflight ile eşzamanlı sorgu birleştirme uygulanır.\n- İçerik yayınlandığında ilgili önbellek anahtarları anında düşürülür.\n- Kompakt profilde bellek sınırı aşılmayacak şekilde bounded cache kullanılır.",
			category:     "Performans",
			author:       "Performans Ekibi",
			coverMediaID: "demo-media-news-6",
			featured:     false,
			tags:         "cache, performance, singleflight, memory, scale",
			body:         `{"ops":[{"insert":"Yüksek Trafik Altında Dayanıklı Önbellekleme\n","attributes":{"header":1}},{"insert":"Cache stampede problemi, aynı anda gelen binlerce isteğin veritabanını kilitlemesine yol açabilir. HydroGo'nun yerleşik önbellek katmanı bu durumu singleflight mekanizması ile tamamen engeller.\n"}]}`,
		},
		{
			id:           "demo-news-theme-engine",
			slug:         "dosya-sistemi-tabanli-tema-motoru-ve-slotlar",
			title:        "Dosya Sistemi Tabanlı Tema Motoru: HTML Şablonları ve Dinamik Slotlar",
			excerpt:      "Harici şablon derleyicilerine ihtiyaç duymadan Go'nun güvenli html/template motoru ile tam özelleştirilebilir tema altyapısı.",
			takeaways:    "- Temalar layouts ve blocks klasör yapısı ile düzenlenir.\n- Dinamik slot mekanizması içerik bloklarının esnek yerleşimini sağlar.\n- Güvenli HTML çıktısı ve CSP uyumu korunur.",
			category:     "Frontend & Tasarım",
			author:       "Tasarım Ekibi",
			coverMediaID: "demo-media-news-7",
			featured:     false,
			tags:         "theme, templates, slots, blocks, frontend",
			body:         `{"ops":[{"insert":"Esnek ve Güvenli Tema Geliştirme\n","attributes":{"header":1}},{"insert":"HydroGo tema motoru, kurumsal markaların kurumsal kimliklerine uygun arayüzler oluşturmasını kolaylaştırır. themes/newsroom klasörü altındaki şablonlar sıfır derleme adımı ile anında güncellenir.\n"}]}`,
		},
		{
			id:           "demo-news-commerce-pilot",
			slug:         "e-ticaret-ve-odeme-saglayici-sozlesmeleri",
			title:        "E-Ticaret Mimarisi: Varyantlar, Stok Rezervasyonu ve Ödeme Entegrasyonları",
			excerpt:      "Stripe, iyzico ve PayTR entegrasyonlarını destekleyen sipariş, sepet ve ödeme provizyon akışları.",
			takeaways:    "- Renk, beden ve lisans bazlı esnek varyant desteği.\n- Stok tükenmesi durumunda sepet seviyesinde koruma.\n- Idempotent sipariş oluşturma ve güvenli ödeme intent akışı.",
			category:     "E-Ticaret",
			author:       "Commerce Ekibi",
			coverMediaID: "demo-media-news-8",
			featured:     false,
			tags:         "commerce, products, variants, payments, shop",
			body:         `{"ops":[{"insert":"Güvenilir ve Esnek E-Ticaret Altyapısı\n","attributes":{"header":1}},{"insert":"Commerce modülü modern e-ticaret operasyonlarının tüm ihtiyaçlarını karşılar. Müşteriler sepete ürün ekleyebilir, varyant seçebilir ve güvenli ödeme adımlarını tamamlayabilir.\n"}]}`,
		},
	}
	if *count < len(newsList) {
		newsList = newsList[:*count]
	} else {
		baseCount := len(newsList)
		for i := baseCount; i < *count; i++ {
			item := newsList[(i-baseCount)%baseCount]
			item.id = fmt.Sprintf("demo-news-generated-%03d", i+1)
			item.slug = fmt.Sprintf("demo-article-%03d", i+1)
			item.title = fmt.Sprintf("%s · Örnek %02d", item.title, i+1)
			item.excerpt = fmt.Sprintf("%s Bu içerik demo kataloğunu göstermek için üretildi.", item.excerpt)
			item.featured = false
			newsList = append(newsList, item)
		}
	}
	if demoLocale == "en" {
		for i := range newsList {
			newsList[i] = localizeDemoNews(newsList[i], i+1)
		}
	}

	for _, item := range newsList {
		featuredInt := 0
		if item.featured {
			featuredInt = 1
		}
		var featuredValue any = featuredInt
		if cfg.Database.Dialect == database.Postgres {
			featuredValue = item.featured
		}
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `INSERT IGNORE INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,cover_media_id,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,'published',1,1,1,0,?,?)`, item.id, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredValue, item.tags, item.body, now, now); err != nil {
				return fmt.Errorf("seed news %s: %w", item.id, err)
			}
		} else if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,cover_media_id,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'published',1,1,1,0,$13,$13) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,cover_media_id,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,'published',1,1,1,0,?13,?13)`, item.id, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredValue, item.tags, item.body, now); err != nil {
			return fmt.Errorf("seed news %s: %w", item.id, err)
		}
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `UPDATE news_items SET locale=?,slug=?,title=?,excerpt=?,key_takeaways=?,category=?,author_name=?,cover_media_id=?,featured=?,tags=?,body=?,status='published',published_revision=revision WHERE id=?`, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredValue, item.tags, item.body, item.id); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE news_items SET locale=$1,slug=$2,title=$3,excerpt=$4,key_takeaways=$5,category=$6,author_name=$7,cover_media_id=$8,featured=$9,tags=$10,body=$11,status='published',published_revision=revision WHERE id=$12`, `UPDATE news_items SET locale=?1,slug=?2,title=?3,excerpt=?4,key_takeaways=?5,category=?6,author_name=?7,cover_media_id=?8,featured=?9,tags=?10,body=?11,status='published',published_revision=revision WHERE id=?12`, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredValue, item.tags, item.body, item.id); err != nil {
			return fmt.Errorf("publish news %s: %w", item.id, err)
		}
	}
	if err := seedDemoTaxonomy(ctx, db, cfg.Database.Dialect, newsList); err != nil {
		return fmt.Errorf("seed demo categories: %w", err)
	}
	if cfg.Database.Dialect == database.MySQL {
		if err := seedExecMySQL(ctx, db, `INSERT IGNORE INTO site_settings (`+"`key`"+`,`+"`value`"+`,is_public,updated_at) VALUES (?,?,1,?)`, "site.title", "HydroGo Demo", now); err != nil {
			return fmt.Errorf("seed setting: %w", err)
		}
	} else if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO site_settings (key,value,is_public,updated_at) VALUES ($1,$2,TRUE,$3) ON CONFLICT (tenant_id,key) DO NOTHING`, `INSERT OR IGNORE INTO site_settings (key,value,is_public,updated_at) VALUES (?,?,1,?)`, "site.title", "HydroGo Demo", now); err != nil {
		return fmt.Errorf("seed setting: %w", err)
	}
	if cfg.Database.Dialect == database.MySQL {
		if err := seedExecMySQL(ctx, db, `DELETE FROM navigation_items WHERE id LIKE ?`, "demo-nav-%"); err != nil {
			return fmt.Errorf("remove legacy demo navigation: %w", err)
		}
	} else if err := seedExec(ctx, db, cfg.Database.Dialect, `DELETE FROM navigation_items WHERE id LIKE $1`, `DELETE FROM navigation_items WHERE id LIKE ?`, "demo-nav-%"); err != nil {
		return fmt.Errorf("remove legacy demo navigation: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "demo-seeded pages=%d news=%d media=%d locale=%s navigation=0\n", 3, len(newsList), len(mediaFixtures), demoLocale)
	return err
}

func cleanDemoCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("clean", flag.ContinueOnError)
	flags.SetOutput(stderr)
	demo := flags.Bool("demo", false, "Remove only HydroGo demo records")
	forceProd := flags.Bool("force-prod", false, "Allow demo cleanup in production")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if !*demo {
		return errors.New("clean requires --demo; no other data is removed")
	}
	if isProductionEnvironment() && !*forceProd {
		return errors.New("demo cleanup is disabled in production; pass --force-prod only for an intentional cleanup")
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	if err := database.CheckCompatibilityForModules(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules)); err != nil {
		return err
	}
	if err := cleanDemoData(ctx, db, cfg.Database.Dialect, cfg.Media.Dir); err != nil {
		return fmt.Errorf("clean demo data: %w", err)
	}
	_, err = fmt.Fprintln(stdout, "demo-data-cleaned")
	return err
}

func isProductionEnvironment() bool {
	for _, key := range []string{"HYDRO_ENV", "APP_ENV", "GO_ENV"} {
		if strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "production") || strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "prod") {
			return true
		}
	}
	return false
}

func cleanDemoData(ctx context.Context, db *sql.DB, dialect database.Dialect, mediaDir string) error {
	if db == nil {
		return errors.New("database is required")
	}
	query := `SELECT storage_key FROM media_assets WHERE id LIKE 'demo-media-%'`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, key := range keys {
		if filepath.Base(key) != key {
			return errors.New("unsafe demo media storage key")
		}
		if err := os.Remove(filepath.Join(mediaDir, key)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, statement := range []string{
		`DELETE FROM page_revisions WHERE page_id LIKE 'demo-%'`,
		`DELETE FROM news_taxonomy_relations WHERE content_id LIKE 'demo-news-%'`,
		`DELETE FROM news_items WHERE id LIKE 'demo-news-%'`,
		`DELETE FROM news_taxonomy_terms WHERE id LIKE 'demo-category-%'`,
		`DELETE FROM pages WHERE id LIKE 'demo-%'`,
		`DELETE FROM members WHERE id='demo-member-portal'`,
		`DELETE FROM media_assets WHERE id LIKE 'demo-media-%'`,
		`DELETE FROM navigation_items WHERE id LIKE 'demo-nav-%'`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func localizeDemoNews(item demoNewsSeed, number int) demoNewsSeed {
	item.slug = fmt.Sprintf("demo-story-%03d", number)
	item.title = fmt.Sprintf("HydroGo Demo Story %02d", number)
	item.excerpt = "A sample article created to demonstrate the HydroGo publishing workflow."
	item.takeaways = "- Sample content for demonstrations\n- Safe to remove with hydro seed --clean"
	item.category = "Demo updates"
	item.author = "HydroGo Demo Team"
	item.tags = "demo, hydrogo, example"
	item.body = "This sample article demonstrates publishing, categories, and search in the HydroGo demo workspace. The content is synthetic and does not describe a customer or commercial offer."
	return item
}

func seedDemoTaxonomy(ctx context.Context, db database.Executor, dialect database.Dialect, items []demoNewsSeed) error {
	if len(items) == 0 {
		return nil
	}
	categories := make(map[string]string)
	for _, item := range items {
		name := strings.TrimSpace(item.category)
		if name != "" {
			categories[name] = "demo-category-" + demoTaxonomySlug(name)
		}
	}
	names := make([]string, 0, len(categories))
	for name := range categories {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		id := categories[name]
		labels, err := json.Marshal(map[string]string{"tr": demoCategoryTurkish(name), "en": name})
		if err != nil {
			return err
		}
		if dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db,
				`INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,'category',NULL,?) ON DUPLICATE KEY UPDATE slug=VALUES(slug),labels_json=VALUES(labels_json)`, id, demoTaxonomySlug(name), string(labels)); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, dialect,
			`INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES ($1,$2,'category',NULL,$3) ON CONFLICT(id) DO UPDATE SET slug=EXCLUDED.slug,labels_json=EXCLUDED.labels_json`,
			`INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,'category',NULL,?) ON CONFLICT(id) DO UPDATE SET slug=excluded.slug,labels_json=excluded.labels_json`, id, demoTaxonomySlug(name), string(labels)); err != nil {
			return err
		}
	}
	for _, item := range items {
		categoryID := categories[strings.TrimSpace(item.category)]
		if categoryID == "" {
			continue
		}
		if dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db,
				`INSERT IGNORE INTO news_taxonomy_relations (content_id,term_id) VALUES (?,?)`, item.id, categoryID); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, dialect,
			`INSERT INTO news_taxonomy_relations (content_id,term_id) VALUES ($1,$2) ON CONFLICT(content_id,term_id) DO NOTHING`,
			`INSERT INTO news_taxonomy_relations (content_id,term_id) VALUES (?,?) ON CONFLICT(content_id,term_id) DO NOTHING`, item.id, categoryID); err != nil {
			return err
		}
	}
	return nil
}

func demoCategoryTurkish(english string) string {
	if english == "Demo updates" {
		return "Demo Güncellemeleri"
	}
	return english
}

func demoTaxonomySlug(value string) string {
	var slug strings.Builder
	separator := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			slug.WriteRune(char)
			separator = false
		case char == ' ' || char == '-' || char == '_' || char == '/' || char == '&':
			if slug.Len() > 0 && !separator {
				slug.WriteByte('-')
				separator = true
			}
		}
	}
	return strings.Trim(slug.String(), "-")
}

func seedExec(ctx context.Context, db database.Executor, dialect database.Dialect, postgresQuery, sqliteQuery string, args ...any) error {
	query := postgresQuery
	if dialect == database.SQLite {
		query = sqliteQuery
	}
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// seedExecMySQL mirrors seedExec for MySQL, which needs its own query text
// (plain, non-numbered "?" placeholders, "INSERT IGNORE"/"ON DUPLICATE KEY
// UPDATE" instead of SQLite/Postgres upsert syntax) and its own arg list,
// since MySQL placeholders cannot be referenced by number and reused the way
// "$7"/"?7" are reused above for a value written to two columns.
func seedExecMySQL(ctx context.Context, db database.Executor, mysqlQuery string, args ...any) error {
	_, err := db.ExecContext(ctx, mysqlQuery, args...)
	return err
}
