package config

import "testing"

func FuzzLoadEnvironmentValues(f *testing.F) {
	for _, seed := range []string{"127.0.0.1:8080", "", "[::1]:8080", "https://example.test", "not an origin"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		_, _ = Load("", lookup(map[string]string{
			"HYDRO_DATABASE_DSN":   "file:fuzz.sqlite",
			"HYDRO_HTTP_ADDRESS":   value,
			"HYDRO_PUBLIC_ORIGIN":  value,
			"HYDRO_DEFAULT_LOCALE": value,
		}), Overrides{})
	})
}
