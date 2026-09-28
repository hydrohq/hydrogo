package database_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestMySQLInitialMigrationChainContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_MYSQL_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_MYSQL_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.MySQL, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var tableCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()`).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Skip("initial migration contract requires an empty dedicated MySQL database")
	}
	ids := []string{
		"202609060001_transaction_contract", "202609060002_auth_security", "202609080003_pages_content",
		"202609080004_media_assets", "202609080005_published_route_index", "202609080006_published_projection",
		"202609080007_mfa", "202609080008_recovery_tokens", "202609080009_site_edition", "202609080010_auth_invites",
		"202609080011_navigation_position_unique", "202609080012_news_items", "202609080013_jobs",
		"202609080014_published_projection_backfill", "202609090015_restore_controls", "202609090016_restore_write_fence",
		"202609090017_audit_request_id", "202609090018_job_retry_state", "202609090019_job_dead_letter_index",
		"202609090020_job_capacity_lock", "202609100001_page_blocks", "202609100002_commerce_core",
		"202609100003_commerce_payments", "202609100004_commerce_carts", "202609100005_commerce_variants",
		"202609100006_outbox_delivery", "202609100007_commerce_order_customer", "202609100008_commerce_order_client_ip",
		"202609100009_commerce_product_description", "202609110001_news_cover_media", "202609110002_commerce_product_media",
		"202609110003_news_editorial_metadata", "202609110004_news_key_takeaways", "202609110005_members",
		"202609110006_members_metadata", "202609120001_media_variants", "202609140001_pages_list_indexes",
		"202609140002_news_list_indexes", "202609160001_news_taxonomy", "202609160002_search_index",
		"202609160003_analytics", "202609160004_forms", "202609170001_form_idempotency",
		"202609170002_commerce_order_idempotency_hash", "202609190001_mobile_device_sessions",
		"202609190002_member_mobile_device_sessions", "202609190003_outbox_tenant_scope",
		"202609190004_audit_tenant_scope", "202609190005_media_tenant_scope", "202609190006_pages_tenant_scope",
		"202609190007_news_tenant_scope", "202609190008_site_tenant_scope", "202609190009_members_tenant_scope",
		"202609190010_commerce_catalog_tenant_scope", "202609190011_commerce_checkout_tenant_scope",
		"202609190012_forms_tenant_scope", "202609200001_auth_identity_tenant_scope",
		"202609200002_auth_security_tenant_scope", "202609200003_admin_user_permissions", "202609200003_postgres_tenant_rls",
		"202609210001_tenant_registry_and_bootstrap", "202609210002_outbox_dead_letters", "202609220001_analytics_content_views",
		"202609270001_forms_mail_crm", "202609280001_seo_redirect_rules", "202609280002_analytics_privacy_stats",
		"202609280003_security_ip_bans", "202609280004_immutable_audit_log", "202609280005_admin_notifications",
		"202609280006_admin_weekly_digest",
	}
	bundle := fstest.MapFS{}
	for _, id := range ids {
		for _, name := range []string{"migration.yaml", "mysql.sql"} {
			content, err := migrations.FS.ReadFile(id + "/" + name)
			if err != nil {
				t.Fatal(err)
			}
			bundle[id+"/"+name] = &fstest.MapFile{Data: content}
		}
	}
	initial := fstest.MapFS{}
	for _, id := range ids[:13] {
		initial[id+"/migration.yaml"] = bundle[id+"/migration.yaml"]
		initial[id+"/mysql.sql"] = bundle[id+"/mysql.sql"]
	}
	if err := database.ApplyMigrations(ctx, db, database.MySQL, initial); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pages (id,title,locale,slug,body,status,created_at,updated_at) VALUES ('backfill','Backfill','en','backfill','published body','published',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO outbox_events (id,topic,aggregate_id,payload_version) VALUES ('outbox1','test','aggregate1',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO site_settings (`key`,`value`,updated_at) VALUES ('legacy.title','Legacy title',CURRENT_TIMESTAMP(6))"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO navigation_items (id,locale,label,href,position,updated_at) VALUES ('legacy-nav','en','Legacy','/legacy',1,CURRENT_TIMESTAMP(6))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO page_redirects (source_path,target_path,status,created_at) VALUES ('/old','/new',301,CURRENT_TIMESTAMP(6))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO audit_events (id,action,target_id) VALUES ('immutable1','test','subject1')`); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrations(ctx, db, database.MySQL, bundle); err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, table := range []string{
			"admin_digest_preferences", "admin_notifications", "security_ip_bans", "analytics_daily_visitors", "analytics_daily_stats",
			"seo_redirect_rules", "mail_spool", "form_lead_notes", "analytics_content_daily", "outbox_dead_letters", "tenant_system_audit", "tenants",
			"admin_user_permissions", "auth_invites", "member_mobile_device_sessions", "mobile_device_sessions", "analytics_daily", "analytics_consent",
			"hydro_forms", "form_inbox", "page_redirects", "navigation_items", "site_settings", "auth_recovery_tokens",
			"news_taxonomy_relations", "news_taxonomy_terms", "hydro_search_documents", "member_sessions", "members",
			"media_variants", "commerce_payment_webhook_events", "commerce_payment_transactions", "commerce_order_items", "commerce_cart_items",
			"commerce_product_variants", "commerce_carts", "commerce_orders", "commerce_products", "job_queue_capacity_lock",
			"restore_controls", "jobs", "news_items", "mfa_recovery_codes",
			"user_mfa", "page_revisions", "sessions", "users", "media_assets", "pages", "audit_events",
			"outbox_events", "schema_migrations",
		} {
			if _, err := db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+table); err != nil {
				t.Errorf("drop %s: %v", table, err)
			}
		}
	}()
	now := time.Now().UTC()
	for _, tenantID := range []string{"tenant-one", "tenant-two"} {
		if _, err := db.ExecContext(ctx, "INSERT INTO site_settings (tenant_id,`key`,`value`,is_public,updated_at) VALUES (?, 'site.title', ?, 1, ?)", tenantID, tenantID, now); err != nil {
			t.Fatalf("tenant site setting insert for %s: %v", tenantID, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO navigation_items (tenant_id,id,locale,label,href,position,updated_at) VALUES (?, ?, 'en', 'Home', '/', 1, ?)`, tenantID, tenantID+"-home", now); err != nil {
			t.Fatalf("tenant navigation insert for %s: %v", tenantID, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO page_redirects (tenant_id,source_path,target_path,status,created_at) VALUES (?, '/same-old', '/same-new', 301, ?)`, tenantID, now); err != nil {
			t.Fatalf("tenant redirect insert for %s: %v", tenantID, err)
		}
	}
	var legacySetting, legacyNavigation, legacyRedirect string
	if err := db.QueryRowContext(ctx, "SELECT `value` FROM site_settings WHERE tenant_id='' AND `key`='legacy.title'").Scan(&legacySetting); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT id FROM navigation_items WHERE tenant_id='' AND id='legacy-nav'`).Scan(&legacyNavigation); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT target_path FROM page_redirects WHERE tenant_id='' AND source_path='/old'`).Scan(&legacyRedirect); err != nil {
		t.Fatal(err)
	}
	if legacySetting != "Legacy title" || legacyNavigation != "legacy-nav" || legacyRedirect != "/new" {
		t.Fatalf("legacy tenant backfill = %q, %q, %q", legacySetting, legacyNavigation, legacyRedirect)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,created_at,updated_at) VALUES ('u1','one@example.test','hash','owner',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,created_at,updated_at) VALUES ('u2','two@example.test','hash','owner',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id,user_id,token_hash,expires_at,created_at,last_seen_at) VALUES ('s1','u1',REPEAT('a',64),?,?,?)`, now.Add(time.Hour), now, now); err != nil {
		t.Fatalf("session insert with parent key: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO user_mfa (user_id,secret_ciphertext,enabled,created_at,updated_at) VALUES ('u1',X'010203',1,CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`); err != nil {
		t.Fatalf("MFA insert with parent key: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO admin_user_permissions (user_id,permission,allowed,created_at,updated_at) VALUES ('u1','pages.publish',1,?,?)`, now, now); err != nil {
		t.Fatalf("admin permission insert with parent key: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO mfa_recovery_codes (id,user_id,code_hash) VALUES ('rc1','u1',REPEAT('b',64))`); err != nil {
		t.Fatalf("MFA recovery insert with parent key: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO auth_recovery_tokens (id,user_id,purpose,token_hash,expires_at,created_at) VALUES ('rt1','u1','password',REPEAT('c',64),CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`); err != nil {
		t.Fatalf("auth recovery insert with parent key: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id='u1'`); err != nil {
		t.Fatalf("delete user with dependent records: %v", err)
	}
	for _, table := range []string{"sessions", "user_mfa", "mfa_recovery_codes", "auth_recovery_tokens", "admin_user_permissions"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("%s rows after parent delete = %d, want 0", table, count)
		}
	}
	for _, id := range []string{"p1", "p2"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO pages (id,title,locale,slug,body,status,created_at,updated_at) VALUES (?, 'Draft','en','same-draft-route', '', 'draft',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`, id); err != nil {
			t.Fatalf("insert duplicate draft route %s: %v", id, err)
		}
	}
	var blocks, publishedBlocks string
	if err := db.QueryRowContext(ctx, `SELECT blocks,published_blocks FROM pages WHERE id='p1'`).Scan(&blocks, &publishedBlocks); err != nil {
		t.Fatal(err)
	}
	if blocks != "[]" || publishedBlocks != "[]" {
		t.Fatalf("old-write block defaults = (%q,%q), want empty JSON arrays", blocks, publishedBlocks)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES ('rev1','p1',1,'en','same-draft-route','Draft','body',?)`, now); err != nil {
		t.Fatalf("page revision insert from old binary shape: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT blocks FROM page_revisions WHERE id='rev1'`).Scan(&blocks); err != nil {
		t.Fatal(err)
	}
	if blocks != "[]" {
		t.Fatalf("page revision blocks = %q, want empty JSON array", blocks)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pages (id,title,locale,slug,body,status,created_at,updated_at) VALUES ('p3','Published','en','about','body','published',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pages (id,title,locale,slug,body,status,created_at,updated_at) VALUES ('p4','Duplicate','en','about','body','published',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`); err == nil {
		t.Fatal("accepted duplicate non-empty locale/slug pair")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,created_at,updated_at) VALUES ('m1','private/m1','photo.png','image/png',3,?,?)`, now, now); err != nil {
		t.Fatalf("media insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO news_items (id,locale,slug,title,body,status,created_at,updated_at) VALUES ('n1','en','news','News','body','draft',?,?)`, now, now); err != nil {
		t.Fatalf("news insert: %v", err)
	}
	var excerpt, tags, keyTakeaways string
	if err := db.QueryRowContext(ctx, `SELECT excerpt,tags,key_takeaways FROM news_items WHERE id='n1'`).Scan(&excerpt, &tags, &keyTakeaways); err != nil {
		t.Fatal(err)
	}
	if excerpt != "" || tags != "" || keyTakeaways != "" {
		t.Fatalf("legacy news defaults = (%q,%q,%q)", excerpt, tags, keyTakeaways)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO members (id,email,password_hash,created_at,updated_at) VALUES ('member1','member@example.test','hash',?,?)`, now, now); err != nil {
		t.Fatalf("member insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO member_sessions (id,member_id,token_hash,expires_at,created_at,last_seen_at) VALUES ('member-session1','member1',REPEAT('d',64),?,?,?)`, now.Add(time.Hour), now, now); err != nil {
		t.Fatalf("member session insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO member_mobile_device_sessions (id,member_id,device_id,family_id,refresh_token_hash,access_token_hash,access_expires_at,refresh_expires_at,created_at,last_used_at) VALUES ('member-mobile1','member1','device1','family1',REPEAT('1',64),REPEAT('2',64),?,?,?,?)`, now.Add(time.Hour), now.Add(24*time.Hour), now, now); err != nil {
		t.Fatalf("member mobile session insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM members WHERE id='member1'`); err != nil {
		t.Fatalf("member cascade delete: %v", err)
	}
	var memberSessionCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM member_sessions`).Scan(&memberSessionCount); err != nil {
		t.Fatal(err)
	}
	var memberMobileSessionCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM member_mobile_device_sessions`).Scan(&memberMobileSessionCount); err != nil {
		t.Fatal(err)
	}
	if memberSessionCount != 0 || memberMobileSessionCount != 0 {
		t.Fatalf("member session rows after delete = (%d,%d), want (0,0)", memberSessionCount, memberMobileSessionCount)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO media_variants (id,media_id,kind,storage_key,mime_type,size_bytes,created_at,updated_at) VALUES ('variant-media1','m1','thumbnail','private/m1/thumb','image/webp',3,?,?)`, now, now); err != nil {
		t.Fatalf("media variant insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO news_taxonomy_terms (id,slug,kind,labels_json) VALUES ('term1','updates','category','{}')`); err != nil {
		t.Fatalf("taxonomy term insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO news_taxonomy_relations (content_id,term_id) VALUES ('n1','term1')`); err != nil {
		t.Fatalf("taxonomy relation insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO hydro_search_documents (id,locale,title,body) VALUES ('search1','en','Search','document')`); err != nil {
		t.Fatalf("search document insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO analytics_consent (consent_id,decision,purpose_version,decided_at) VALUES ('consent1','granted','v1',?)`, now); err != nil {
		t.Fatalf("analytics consent insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO analytics_daily (day,event_type,country,purpose_version,event_count,duration_seconds) VALUES ('2026-09-28','page_view','TR','v1',1,4)`); err != nil {
		t.Fatalf("analytics daily insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO hydro_forms (id,title,schema_json,updated_at) VALUES ('contact','Contact','{}',?)`, now); err != nil {
		t.Fatalf("form schema insert: %v", err)
	}
	for i := range 2 {
		id := fmt.Sprintf("submission-empty-%d", i)
		if _, err := db.ExecContext(ctx, `INSERT INTO form_inbox (id,form_key,email,payload,source_ip_hash,created_at) VALUES (?,?,?,?,?,?)`, id, "contact", "sender@example.test", "{}", "", now); err != nil {
			t.Fatalf("insert empty-idempotency submission: %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO form_inbox (id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash) VALUES ('submission-key1','contact','sender@example.test','{}','', 'new',?,'key1',REPEAT('e',64))`, now); err != nil {
		t.Fatalf("insert idempotent submission: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO form_inbox (id,form_key,email,payload,source_ip_hash,status,created_at,idempotency_key,request_hash) VALUES ('submission-key2','contact','sender@example.test','{}','', 'new',?,'key1',REPEAT('e',64))`, now); err == nil {
		t.Fatal("accepted duplicate non-empty form idempotency key")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO form_lead_notes (id,tenant_id,submission_id,author_id,note,created_at) VALUES ('note1','','submission-empty-0','u1','Follow-up',?)`, now); err != nil {
		t.Fatalf("lead note insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO form_lead_notes (id,tenant_id,submission_id,author_id,note,created_at) VALUES ('note-cross','tenant-two','submission-empty-0','u1','Cross tenant',?)`, now); err == nil {
		t.Fatal("accepted cross-tenant lead note")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO mail_spool (id,event_key,recipient,subject,body,html_body,next_attempt_at,created_at,updated_at) VALUES ('mail1','contact-receipt','sender@example.test','Received','Text','<p>Received</p>',?,?,?)`, now, now, now); err != nil {
		t.Fatalf("mail spool insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO outbox_dead_letters (id,topic,aggregate_id,payload_version,attempts,last_error,failed_at) VALUES ('dead1','mail','aggregate1',1,5,'delivery failed',?)`, now); err != nil {
		t.Fatalf("outbox dead letter insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO seo_redirect_rules (tenant_id,id,pattern,match_kind,target_path,status,created_at) VALUES ('tenant-one','r1','/legacy','exact','/home',301,?)`, now); err != nil {
		t.Fatalf("SEO redirect rule insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO analytics_daily_stats (day,tenant_id,metric,dimension,event_count) VALUES ('2026-09-28','tenant-one','page','/home',1)`); err != nil {
		t.Fatalf("analytics privacy stats insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO analytics_daily_visitors (day,tenant_id,visitor_key) VALUES ('2026-09-28','tenant-one',REPEAT('a',64))`); err != nil {
		t.Fatalf("analytics visitor insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO analytics_content_daily (day,tenant_id,content_type,content_id,purpose_version,view_count) VALUES ('2026-09-28','tenant-one','page','page1','v1',1)`); err != nil {
		t.Fatalf("analytics content view insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO security_ip_bans (address,expires_at,created_at) VALUES ('192.0.2.1',?,?)`, now.Add(time.Hour), now); err != nil {
		t.Fatalf("security IP ban insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE audit_events SET action='changed' WHERE id='immutable1'`); err == nil {
		t.Fatal("updated immutable audit record")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM audit_events WHERE id='immutable1'`); err == nil {
		t.Fatal("deleted immutable audit record")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO admin_notifications (id,tenant_id,user_id,title,message,created_at) VALUES ('notice1','tenant-one','u1','Update','Content updated',?)`, now); err != nil {
		t.Fatalf("admin notification insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO admin_digest_preferences (tenant_id,user_id,enabled,timezone) VALUES ('tenant-one','u1',1,'Europe/Istanbul')`); err != nil {
		t.Fatalf("admin digest preference insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO mobile_device_sessions (id,user_id,device_id,family_id,refresh_token_hash,access_token_hash,access_expires_at,refresh_expires_at,created_at,last_used_at) VALUES ('mobile1','u2','device1','family1',REPEAT('f',64),REPEAT('0',64),?,?,?,?)`, now.Add(time.Hour), now.Add(24*time.Hour), now, now); err != nil {
		t.Fatalf("mobile session insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id='u2'`); err != nil {
		t.Fatalf("mobile session parent delete: %v", err)
	}
	var mobileSessions int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mobile_device_sessions`).Scan(&mobileSessions); err != nil {
		t.Fatal(err)
	}
	if mobileSessions != 0 {
		t.Fatalf("mobile sessions after user delete = %d, want 0", mobileSessions)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO jobs (id,type,payload,available_at,created_at) VALUES ('j1','test','{}',?,?)`, now, now); err != nil {
		t.Fatalf("job insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO job_queue_capacity_lock (id) VALUES (1) ON DUPLICATE KEY UPDATE id=id`); err != nil {
		t.Fatalf("job capacity lock idempotent insert: %v", err)
	}
	var publishedRevision int
	var publishedSlug, publishedBody string
	if err := db.QueryRowContext(ctx, `SELECT published_revision,published_slug,published_body FROM pages WHERE id='backfill'`).Scan(&publishedRevision, &publishedSlug, &publishedBody); err != nil {
		t.Fatal(err)
	}
	if publishedRevision != 1 || publishedSlug != "backfill" || publishedBody != "published body" {
		t.Fatalf("published projection = (%d,%q,%q)", publishedRevision, publishedSlug, publishedBody)
	}
	var availableAt time.Time
	if err := db.QueryRowContext(ctx, `SELECT available_at FROM outbox_events WHERE id='outbox1'`).Scan(&availableAt); err != nil {
		t.Fatal(err)
	}
	if availableAt.IsZero() {
		t.Fatal("outbox availability backfill is zero")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO commerce_products (id,sku,title,price_cents,currency,stock,status,created_at,updated_at,description) VALUES ('product1','sku1','Product',100,'USD',5,'active',?,?,'A description')`, now, now); err != nil {
		t.Fatalf("product insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO commerce_orders (id,user_id,idempotency_key,currency,total_cents,payment_status,shipping_status,created_at) VALUES ('order1','u2','key1','USD',100,'pending','pending',?)`, now); err != nil {
		t.Fatalf("order insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO commerce_order_items (id,order_id,product_id,sku,title,quantity,unit_cents,total_cents) VALUES ('item1','order1','product1','sku1','Product',1,100,100)`); err != nil {
		t.Fatalf("order item insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO commerce_product_variants (id,product_id,sku,title,price_cents,stock,status,created_at,updated_at) VALUES ('variant1','product1','sku1-blue','Blue',100,5,'active',?,?)`, now, now); err != nil {
		t.Fatalf("variant insert: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE commerce_order_items SET variant_id='variant1' WHERE id='item1'`); err != nil {
		t.Fatalf("variant assignment: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO restore_controls (id,sessions_revoked,egress_quarantined,old_writes_blocked,activated_at) VALUES (1,0,0,1,?)`, now); err != nil {
		t.Fatal(err)
	}
	for name, query := range map[string]string{
		"page insert":  `INSERT INTO pages (id,title,locale,slug,body,status,created_at,updated_at) VALUES ('blocked','Blocked','en','blocked','body','draft',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`,
		"page update":  `UPDATE pages SET title='blocked' WHERE id='p1'`,
		"page delete":  `DELETE FROM pages WHERE id='p1'`,
		"media insert": `INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,created_at,updated_at) VALUES ('blocked','private/blocked','photo.png','image/png',3,CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`,
		"news insert":  `INSERT INTO news_items (id,locale,slug,title,body,status,created_at,updated_at) VALUES ('blocked','en','blocked','Blocked','body','draft',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`,
		"job insert":   `INSERT INTO jobs (id,type,payload,available_at,created_at) VALUES ('blocked','test','{}',CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))`,
	} {
		if _, err := db.ExecContext(ctx, query); err == nil {
			t.Errorf("restore fence accepted %s", name)
		}
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM restore_controls WHERE id=1`); err != nil {
		t.Fatal(err)
	}
}

func TestMySQLPoolAndTransactionContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_MYSQL_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_MYSQL_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.MySQL, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := db.Stats().MaxOpenConnections; got != 10 {
		t.Fatalf("default MySQL pool max = %d, want 10", got)
	}

	bounded, err := database.OpenWithOptions(ctx, database.MySQL, dsn, database.OpenOptions{MaxOpenConns: 4, MaxIdleConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer bounded.Close()
	if got := bounded.Stats().MaxOpenConnections; got != 4 {
		t.Fatalf("configured MySQL pool max = %d, want 4", got)
	}
	name := fmt.Sprintf("hydro_mysql_tx_%d", os.Getpid())
	if _, err := bounded.ExecContext(ctx, "CREATE TABLE "+name+" (id INT PRIMARY KEY) ENGINE=InnoDB"); err != nil {
		t.Fatal(err)
	}
	defer bounded.ExecContext(context.Background(), "DROP TABLE "+name)
	wantErr := errors.New("rollback contract")
	err = database.NewTransactor(bounded).Within(ctx, func(tx database.Executor) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO "+name+" (id) VALUES (?)", 1); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("transaction error = %v, want %v", err, wantErr)
	}
	var count int
	if err := bounded.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+name).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rows after rollback = %d, want 0", count)
	}
}

func TestMySQLNavigationMigrationContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_MYSQL_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_MYSQL_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.MySQL, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var ledgerExists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations'`).Scan(&ledgerExists); err != nil {
		t.Fatal(err)
	}
	if ledgerExists != 0 {
		t.Skip("navigation migration contract requires an empty dedicated MySQL database")
	}
	table := fmt.Sprintf("hydro_mysql_navigation_%d", os.Getpid())
	baseTable := table + "_base"
	if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (id VARCHAR(128) PRIMARY KEY) ENGINE=InnoDB"); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+table)
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS schema_migrations")
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+baseTable)
	metadata, err := migrations.FS.ReadFile("202609280023_navigation_tree/migration.yaml")
	if err != nil {
		t.Fatal(err)
	}
	script, err := migrations.FS.ReadFile("202609280023_navigation_tree/mysql.sql")
	if err != nil {
		t.Fatal(err)
	}
	script = []byte(strings.ReplaceAll(string(script), "navigation_items", table))
	bundle := fstest.MapFS{
		"000000000001_mysql_contract/migration.yaml":  &fstest.MapFile{Data: []byte("module: core\ntransactional:\n  mysql: false\n")},
		"000000000001_mysql_contract/mysql.sql":       &fstest.MapFile{Data: []byte("CREATE TABLE " + baseTable + " (id VARCHAR(64) PRIMARY KEY) ENGINE=InnoDB")},
		"202609280023_navigation_tree/migration.yaml": &fstest.MapFile{Data: metadata},
		"202609280023_navigation_tree/mysql.sql":      &fstest.MapFile{Data: script},
	}
	for range 2 {
		if err := database.ApplyMigrations(ctx, db, database.MySQL, bundle); err != nil {
			t.Fatal(err)
		}
	}
	for column, length := range map[string]int{"parent_id": 128, "target": 8} {
		var dataType string
		var actualLength int
		if err := db.QueryRowContext(ctx, `SELECT data_type,character_maximum_length FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?`, table, column).Scan(&dataType, &actualLength); err != nil {
			t.Fatal(err)
		}
		if dataType != "varchar" || actualLength != length {
			t.Errorf("%s type = %s(%d), want varchar(%d)", column, dataType, actualLength, length)
		}
	}
}

func TestMySQLSchemaDiffFindsPhysicalDrift(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_MYSQL_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_MYSQL_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.MySQL, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var tableCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()`).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Skip("schema diff contract requires an empty dedicated MySQL database")
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE parent (id VARCHAR(64) PRIMARY KEY, name VARCHAR(64) NOT NULL) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE child (id VARCHAR(64) PRIMARY KEY, parent_id VARCHAR(64) NOT NULL) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS child")
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS parent")
	bundle := fstest.MapFS{
		"001_reference/migration.yaml": &fstest.MapFile{Data: []byte("module: core\nclass:\n  mysql: expand\ntransactional:\n  mysql: false\n")},
		"001_reference/mysql.sql":      &fstest.MapFile{Data: []byte("CREATE TABLE parent (id VARCHAR(64) PRIMARY KEY, name VARCHAR(128) NOT NULL) ENGINE=InnoDB; CREATE INDEX idx_parent_name ON parent(name); CREATE TABLE child (id VARCHAR(64) PRIMARY KEY, parent_id VARCHAR(64) NOT NULL, CONSTRAINT fk_child_parent FOREIGN KEY (parent_id) REFERENCES parent(id)) ENGINE=InnoDB")},
	}
	changes, err := database.MySQLSchemaDiff(ctx, db, bundle, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"definition_mismatch:column:parent.name":   false,
		"missing:index:parent.idx_parent_name":     false,
		"missing:constraint:child.fk_child_parent": false,
	}
	for _, change := range changes {
		key := change.Kind + ":" + change.Object + ":" + change.Name
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("schema diff missing %s; got %#v", key, changes)
		}
	}
}

func TestMySQLMigrationRunnerContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_MYSQL_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_MYSQL_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.MySQL, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var ledgerExists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations'`).Scan(&ledgerExists); err != nil {
		t.Fatal(err)
	}
	if ledgerExists != 0 {
		t.Skip("migration runner contract requires an empty dedicated MySQL database")
	}
	table := fmt.Sprintf("hydro_mysql_migration_%d", os.Getpid())
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+table)
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS schema_migrations")
	bundle := fstest.MapFS{
		"000000000001_mysql_contract/migration.yaml": &fstest.MapFile{Data: []byte("module: core\ntransactional:\n  mysql: false\n")},
		"000000000001_mysql_contract/mysql.sql":      &fstest.MapFile{Data: []byte("CREATE TABLE " + table + " (id VARCHAR(64) PRIMARY KEY) ENGINE=InnoDB; CREATE INDEX " + table + "_id_idx ON " + table + " (id);")},
	}
	for range 2 {
		if err := database.ApplyMigrations(ctx, db, database.MySQL, bundle); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := database.PendingMigrations(ctx, db, database.MySQL, bundle, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending MySQL migrations = %v, want none", pending)
	}
	if err := database.CheckCompatibility(ctx, db, database.MySQL, bundle); err != nil {
		t.Fatalf("MySQL migration compatibility: %v", err)
	}
}
