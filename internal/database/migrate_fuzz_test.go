package database

import (
	"testing"
	testingfst "testing/fstest"
)

func FuzzDiscoverMigrations(f *testing.F) {
	for _, seed := range []string{"001_init.sql", "", "../escape.sql", "999_feature.sql"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if name == "" || name == "." || name == ".." {
			name = "001_init.sql"
		}
		_, _ = discoverMigrations(testingfst.MapFS{name: &testingfst.MapFile{Data: []byte("-- test")}})
	})
}
