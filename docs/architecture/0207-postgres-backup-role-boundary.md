# ADR-0207: Privileged PostgreSQL Backup Role Boundary

Status: Accepted  
Date: 2026-09-21

`FORCE ROW LEVEL SECURITY` is a deliberate tenant-isolation control and must
also protect ordinary application/database-owner connections. PostgreSQL
backup therefore uses a separately provisioned operator role. The role is
passed explicitly through `HYDRO_PG_PRIVILEGED_ROLE` and the backup adapter's
`--role` argument. HydroGo does not contain credentials, grant privileges, or
fall back to an unprivileged dump that could produce a partial snapshot.
