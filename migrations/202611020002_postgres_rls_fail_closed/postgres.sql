DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'pages',
        'page_revisions',
        'audit_events',
        'outbox_events',
        'users',
        'sessions',
        'media_assets',
        'media_variants',
        'user_mfa',
        'mfa_recovery_codes',
        'auth_recovery_tokens',
        'auth_invites',
        'site_settings',
        'navigation_items',
        'page_redirects',
        'form_inbox',
        'news_items',
        'news_taxonomy_terms',
        'news_taxonomy_relations',
        'members',
        'member_sessions',
        'member_mobile_device_sessions',
        'commerce_stock',
        'commerce_shopping_carts',
        'commerce_shopping_cart_items',
        'commerce_customer_orders',
        'commerce_customer_order_items',
        'commerce_payment_charges',
        'commerce_payment_notifications',
        'hydro_forms',
        'hydro_collections',
        'hydro_collection_records',
        'hydro_collection_unique',
        'hydro_collection_relations',
        'hydro_collection_revisions',
        'hydro_collection_translations',
        'hydro_collection_sequences',
        'hydro_roles',
        'hydro_role_permissions',
        'admin_user_permissions',
        'api_tokens'
    ] LOOP
        IF to_regclass(table_name) IS NULL THEN
            CONTINUE;
        END IF;
        EXECUTE 'ALTER TABLE ' || quote_ident(table_name) || ' ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE ' || quote_ident(table_name) || ' FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS hydrogo_tenant_isolation ON ' || quote_ident(table_name);
        EXECUTE 'CREATE POLICY hydrogo_tenant_isolation ON ' || quote_ident(table_name) ||
            ' FOR ALL ' ||
            ' USING (tenant_id = nullif(current_setting(''app.current_tenant_id'', true), ''''))' ||
            ' WITH CHECK (tenant_id = nullif(current_setting(''app.current_tenant_id'', true), ''''))';
    END LOOP;
END $$;
