-- RLS is opt-in per transaction so legacy single-tenant PostgreSQL databases
-- remain compatible. Shared-mode transaction boundaries set
-- app.tenant_rls_enforced=true and app.current_tenant_id before any query.
DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'pages', 'page_revisions', 'audit_events', 'outbox_events',
        'users', 'sessions', 'media_assets', 'media_variants',
        'user_mfa', 'mfa_recovery_codes', 'auth_recovery_tokens',
        'auth_invites', 'site_settings', 'navigation_items',
        'page_redirects', 'form_inbox', 'news_items', 'members',
        'member_sessions', 'member_mobile_device_sessions',
        'commerce_products', 'commerce_product_variants',
        'commerce_orders', 'commerce_order_items', 'commerce_carts',
        'commerce_cart_items', 'commerce_payment_transactions',
        'commerce_payment_webhook_events', 'hydro_forms'
    ] LOOP
        IF to_regclass(table_name) IS NULL THEN
            CONTINUE;
        END IF;
        EXECUTE 'ALTER TABLE ' || quote_ident(table_name) || ' ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE ' || quote_ident(table_name) || ' FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS hydrogo_tenant_isolation ON ' || quote_ident(table_name);
        EXECUTE 'CREATE POLICY hydrogo_tenant_isolation ON ' || quote_ident(table_name) ||
            ' USING ((coalesce(nullif(current_setting(''app.tenant_rls_enforced'', true), ''''), ''false'') <> ''true'') OR tenant_id = nullif(current_setting(''app.current_tenant_id'', true), ''''))' ||
            ' WITH CHECK ((coalesce(nullif(current_setting(''app.tenant_rls_enforced'', true), ''''), ''false'') <> ''true'') OR tenant_id = nullif(current_setting(''app.current_tenant_id'', true), ''''))';
    END LOOP;
END $$;
