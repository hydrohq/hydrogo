package v1

import "testing"

func TestValidateDependencyGraph(t *testing.T) {
	tests := []struct {
		name  string
		graph map[string][]string
		want  bool
	}{
		{"acyclic", map[string][]string{" Commerce ": {"Pages"}, "pages": {}}, true},
		{"unknown dependency", map[string][]string{"commerce": {"missing"}}, false},
		{"cycle", map[string][]string{"a": {"b"}, "b": {"a"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDependencyGraph(tt.graph)
			if (err == nil) != tt.want {
				t.Fatalf("ValidateDependencyGraph() error = %v, want success=%v", err, tt.want)
			}
		})
	}
}
