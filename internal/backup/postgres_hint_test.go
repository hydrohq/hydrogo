package backup

import (
	"strings"
	"testing"
)

func TestPrivilegeHintOnlyWhenARoleWouldHelp(t *testing.T) {
	rls := `pg_dump: error: query failed: ERROR:  query would be affected by row-level security policy for table "audit_chain"`
	if !strings.Contains(privilegeHint(rls, ""), "HYDRO_PG_PRIVILEGED_ROLE") {
		t.Fatal("a row-level security refusal must point at HYDRO_PG_PRIVILEGED_ROLE")
	}
	if privilegeHint(rls, "hydro_backup") != "" {
		t.Fatal("no hint once a privileged role is already set")
	}
	if privilegeHint("pg_dump: error: connection refused", "") != "" {
		t.Fatal("unrelated failures get no hint")
	}
}
