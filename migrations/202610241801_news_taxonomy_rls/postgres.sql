-- news_taxonomy_terms/news_taxonomy_relations were never added to
-- 202609200003_postgres_tenant_rls's table list (migrations are never
-- edited once released, see AGENTS.md); this closes that gap directly.
DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'news_taxonomy_terms', 'news_taxonomy_relations'
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
