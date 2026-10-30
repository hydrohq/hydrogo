DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'hydro_collections',
        'hydro_collection_records',
        'hydro_collection_unique',
        'hydro_collection_relations',
        'hydro_collection_revisions',
        'hydro_collection_translations',
        'hydro_collection_sequences'
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
