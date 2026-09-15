package sdkdist

import "testing"

func TestArtifactNameCanonicalizesSupportedTargets(t *testing.T) {
	for _, test := range []struct {
		os, arch, want string
	}{
		{"linux", "amd64", "hydro-linux-amd64"},
		{"WINDOWS", "ARM64", "hydro-windows-arm64.exe"},
		{"darwin", "arm64", "hydro-darwin-arm64"},
	} {
		got, err := ArtifactName("hydro", Target{OS: test.os, Arch: test.arch})
		if err != nil || got != test.want {
			t.Fatalf("ArtifactName(%q,%q) = %q/%v, want %q", test.os, test.arch, got, err, test.want)
		}
	}
}

func TestArtifactNameRejectsUnsafeOrUnsupportedTargets(t *testing.T) {
	for _, test := range []struct {
		product, os, arch string
	}{
		{"../hydro", "linux", "amd64"},
		{"hydro", "freebsd", "amd64"},
		{"hydro", "linux", "386"},
	} {
		if _, err := ArtifactName(test.product, Target{OS: test.os, Arch: test.arch}); err == nil {
			t.Fatalf("unsafe/unsupported target accepted: %#v", test)
		}
	}
}
