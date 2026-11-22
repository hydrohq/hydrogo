package config

import (
	"strings"
	"testing"
)

type widgetSettings struct {
	Size  int    `yaml:"size"`
	Label string `yaml:"label"`
}

func registerWidgets(t *testing.T) {
	t.Helper()
	RegisterExtension(Extension{
		Section:  "widgets",
		Defaults: func() any { return &widgetSettings{Size: 3} },
		Env: func(lookup LookupEnv, v any) error {
			if value, ok := lookup("HYDRO_WIDGETS_LABEL"); ok {
				v.(*widgetSettings).Label = value
			}
			return nil
		},
		Validate: func(_ Config, v any) error {
			if v.(*widgetSettings).Size > 10 {
				return errTooBig
			}
			return nil
		},
		ProductionIssues: func(_ Config, _ LookupEnv, v any) []string {
			if IsInsecureSecret(v.(*widgetSettings).Label) {
				return []string{"widgets.label is empty or set to CHANGE_ME"}
			}
			return nil
		},
	})
	t.Cleanup(func() {
		extensionsMu.Lock()
		extensions = nil
		extensionsMu.Unlock()
	})
}

type extErr string

func (e extErr) Error() string { return string(e) }

const errTooBig = extErr("widgets.size must be at most 10")

func TestExtensionSectionDefaultsYAMLEnvAndValidation(t *testing.T) {
	registerWidgets(t)
	env := map[string]string{"HYDRO_DATABASE_DSN": "file:ext.sqlite"}

	cfg, err := Load("", lookup(env), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	w, ok := ExtensionValue[*widgetSettings](cfg, "widgets")
	if !ok || w.Size != 3 {
		t.Fatalf("defaults: %+v ok=%v", w, ok)
	}

	path := writeConfig(t, "widgets:\n  size: 7\n  label: from-yaml\n")
	env["HYDRO_WIDGETS_LABEL"] = "from-env"
	cfg, err = Load(path, lookup(env), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	w, _ = ExtensionValue[*widgetSettings](cfg, "widgets")
	if w.Size != 7 || w.Label != "from-env" {
		t.Fatalf("yaml then env: %+v", w)
	}

	path = writeConfig(t, "widgets:\n  size: 11\n")
	if _, err = Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:ext2.sqlite"}), Overrides{}); err == nil || !strings.Contains(err.Error(), "widgets.size") {
		t.Fatalf("expected extension validation error, got %v", err)
	}

	cfg, err = Load(writeConfig(t, "widgets:\n  size: 2\n"), lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:ext5.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if issues := ExtensionProductionIssues(cfg, lookup(nil)); len(issues) != 1 || !strings.Contains(issues[0], "widgets.label") {
		t.Fatalf("production issues: %v", issues)
	}
}

func TestExtensionSectionStrictness(t *testing.T) {
	registerWidgets(t)
	dsn := lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:ext3.sqlite"})

	// A key nobody owns is rejected, naming the key like the strict decoder does.
	if _, err := Load(writeConfig(t, "gadgets:\n  size: 1\n"), dsn, Overrides{}); err == nil || !strings.Contains(err.Error(), "field gadgets not found") {
		t.Fatalf("unknown section: %v", err)
	}
	// Unknown keys inside an owned section are rejected too.
	if _, err := Load(writeConfig(t, "widgets:\n  colour: red\n"), dsn, Overrides{}); err == nil || !strings.Contains(err.Error(), "colour") {
		t.Fatalf("unknown key in section: %v", err)
	}
}

func TestSectionIsUnknownWithoutItsExtension(t *testing.T) {
	// With no extension registered, the section behaves like any unknown key.
	if _, err := Load(writeConfig(t, "widgets:\n  size: 1\n"), lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:ext4.sqlite"}), Overrides{}); err == nil || !strings.Contains(err.Error(), "field widgets not found") {
		t.Fatalf("expected unknown section error, got %v", err)
	}
}
