DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (
        SELECT tablename
        FROM pg_policies
        WHERE schemaname = current_schema()
          AND policyname = 'hydrogo_tenant_isolation'
    ) LOOP
        EXECUTE 'ALTER TABLE ' || quote_ident(r.tablename) || ' ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE ' || quote_ident(r.tablename) || ' FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS hydrogo_tenant_isolation ON ' || quote_ident(r.tablename);
        EXECUTE 'CREATE POLICY hydrogo_tenant_isolation ON ' || quote_ident(r.tablename) ||
            ' FOR ALL ' ||
            ' USING (tenant_id = coalesce(current_setting(''app.current_tenant_id'', true), ''''))' ||
            ' WITH CHECK (tenant_id = coalesce(current_setting(''app.current_tenant_id'', true), ''''))';
    END LOOP;
END $$;
