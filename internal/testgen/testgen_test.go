package testgen

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

func definition(id string, perms contenttypes.Permissions) contenttypes.Definition {
	return contenttypes.Definition{
		ID:          id,
		Label:       id,
		Permissions: perms,
		Fields:      []contenttypes.Field{{ID: "name", Label: "Name", Kind: "text", Required: true}},
	}
}

func TestGenerateIncludesSharedHelperOnce(t *testing.T) {
	files := Generate([]contenttypes.Definition{definition("faq", contenttypes.Permissions{})})
	var helpers int
	for _, f := range files {
		if f.Path == "content/_helpers.ts" {
			helpers++
		}
	}
	if helpers != 1 {
		t.Fatalf("expected exactly one _helpers.ts, got %d", helpers)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (helpers + 1 spec), got %d", len(files))
	}
}

func TestGenerateIsDeterministicAndSortedByID(t *testing.T) {
	defs := []contenttypes.Definition{definition("zebra", contenttypes.Permissions{}), definition("alpha", contenttypes.Permissions{})}
	first := Generate(defs)
	second := Generate(defs)
	if len(first) != len(second) {
		t.Fatalf("non-deterministic file count: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("non-deterministic output at index %d", i)
		}
	}
	if first[1].Path != "content/alpha.spec.ts" || first[2].Path != "content/zebra.spec.ts" {
		t.Fatalf("expected specs sorted by id, got %s then %s", first[1].Path, first[2].Path)
	}
}

func TestGenericPermissionMatrix(t *testing.T) {
	files := Generate([]contenttypes.Definition{definition("faq", contenttypes.Permissions{})})
	spec := specFor(t, files, "faq")

	// Generic read: every built-in role passes.
	for _, role := range []string{"admin", "editor", "viewer"} {
		assertTest(t, spec, role+" can read", "toBe(200)")
	}
	// Generic write: admin and editor pass, viewer is denied.
	assertTest(t, spec, "admin can create", "not.toBe(403)")
	assertTest(t, spec, "editor can create", "not.toBe(403)")
	assertTest(t, spec, "viewer cannot create", "toBe(403)")
}

func TestCustomPermissionMatrix(t *testing.T) {
	files := Generate([]contenttypes.Definition{definition("invoices", contenttypes.Permissions{Read: "erp.invoices.read", Write: "erp.invoices.write"})})
	spec := specFor(t, files, "invoices")

	assertTest(t, spec, "admin can read", "toBe(200)")
	assertTest(t, spec, "editor cannot read", "toBe(403)")
	assertTest(t, spec, "viewer cannot read", "toBe(403)")
	assertTest(t, spec, "admin can create", "not.toBe(403)")
	assertTest(t, spec, "editor cannot create", "toBe(403)")
	assertTest(t, spec, "viewer cannot create", "toBe(403)")
}

func TestGeneratedSpecReferencesTheRightPathAndHelpers(t *testing.T) {
	files := Generate([]contenttypes.Definition{definition("deals", contenttypes.Permissions{})})
	spec := specFor(t, files, "deals")
	if !strings.Contains(spec, `PATH = "/api/v1/admin/content/deals"`) {
		t.Fatalf("spec does not target the right admin API path:\n%s", spec)
	}
	if !strings.Contains(spec, `from "./_helpers"`) {
		t.Fatalf("spec does not import the shared helper:\n%s", spec)
	}
	if !strings.HasPrefix(spec, generatedHeader) {
		t.Fatalf("spec is missing the DO NOT EDIT header:\n%s", spec)
	}
}

func TestHelpersFileHasDoNotEditHeader(t *testing.T) {
	files := Generate([]contenttypes.Definition{definition("faq", contenttypes.Permissions{})})
	for _, f := range files {
		if f.Path == "content/_helpers.ts" {
			if !strings.HasPrefix(f.Content, generatedHeader) {
				t.Fatalf("_helpers.ts is missing the DO NOT EDIT header:\n%s", f.Content)
			}
			return
		}
	}
	t.Fatal("_helpers.ts not generated")
}

func specFor(t *testing.T, files []File, id string) string {
	t.Helper()
	for _, f := range files {
		if f.Path == "content/"+id+".spec.ts" {
			return f.Content
		}
	}
	t.Fatalf("no generated spec for %s", id)
	return ""
}

// assertTest checks that a test named "<name>" exists in spec and that its
// body contains the given status assertion fragment.
func assertTest(t *testing.T, spec, name, statusFragment string) {
	t.Helper()
	marker := `test("` + name + `"`
	idx := strings.Index(spec, marker)
	if idx == -1 {
		t.Fatalf("spec is missing test %q:\n%s", name, spec)
	}
	end := strings.Index(spec[idx:], "  });\n")
	if end == -1 {
		t.Fatalf("test %q body is not closed", name)
	}
	body := spec[idx : idx+end]
	if !strings.Contains(body, statusFragment) {
		t.Fatalf("test %q does not assert %q:\n%s", name, statusFragment, body)
	}
}
