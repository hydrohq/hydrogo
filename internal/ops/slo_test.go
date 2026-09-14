package ops

import (
	"reflect"
	"testing"
)

func TestParseSLOConfigIsTargetNeutral(t *testing.T) {
	config, err := ParseSLOConfig([]byte("objectives:\n  - name: api-availability\n    indicator: http.server.request.success\n    window: 30d\n"))
	if err != nil {
		t.Fatal(err)
	}
	if config.Objectives[0].Target != nil {
		t.Fatal("target should remain optional")
	}
}

func TestValidateSLOConfigRejectsUnsafeOrAmbiguousValues(t *testing.T) {
	for _, config := range []SLOConfig{
		{Objectives: []SLO{{Name: "Bad Name", Indicator: "x", Window: "30d"}}},
		{Objectives: []SLO{{Name: "api", Indicator: "x", Window: "0d"}}},
		{Objectives: []SLO{{Name: "api", Indicator: "x", Window: "30d"}, {Name: "api", Indicator: "y", Window: "30d"}}},
	} {
		if err := ValidateSLOConfig(config); err == nil {
			t.Fatalf("invalid config accepted: %#v", config)
		}
	}
}

func TestSortedObjectives(t *testing.T) {
	config := SLOConfig{Objectives: []SLO{{Name: "zeta"}, {Name: "alpha"}}}
	got := config.SortedObjectives()
	if want := []string{"alpha", "zeta"}; !reflect.DeepEqual([]string{got[0].Name, got[1].Name}, want) {
		t.Fatalf("sorted objectives = %#v", got)
	}
}
