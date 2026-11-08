package storefront

import (
	"bytes"
	"html/template"
	"net/http"
	"sync"
)

var (
	authTmplOnce sync.Once
	authTmpl     *template.Template
	authTmplErr  error
)

// AuthData represents context for dedicated /kayit and /giris customer authentication screens.
type AuthData struct {
	StoreName    string
	Slogan       string
	PrimaryColor string
	BGTone       string
	CleanPhone   string
	ContactEmail string
	FontPreset   string
	FontHeading  template.CSS
	FontBody     template.CSS
	AuthMode     string // "register" or "login"
}

func initAuthTemplate() {
	authTmplOnce.Do(func() {
		funcs := template.FuncMap{
			"add": func(a, b any) float64 {
				toFloat := func(x any) float64 {
					switch v := x.(type) {
					case int:
						return float64(v)
					case int64:
						return float64(v)
					case float64:
						return v
					case float32:
						return float64(v)
					default:
						return 0
					}
				}
				return toFloat(a) + toFloat(b)
			},
			"urlquery": func(s string) string { return template.URLQueryEscaper(s) },
		}
		authTmpl, authTmplErr = template.New("auth_view.html").Funcs(funcs).ParseFS(templateFS, "auth_view.html", "sections/*.html")
	})
}

// RenderAuth renders dedicated customer registration or login screen.
func RenderAuth(w http.ResponseWriter, r *http.Request, data AuthData) error {
	initAuthTemplate()
	if authTmplErr != nil {
		return authTmplErr
	}
	if data.StoreName == "" {
		data.StoreName = "Atelier Riviera"
	}
	if data.PrimaryColor == "" {
		data.PrimaryColor = "#8C6D46"
	}
	if data.BGTone == "" {
		data.BGTone = "#FAF8F5"
	}
	if data.FontHeading == "" || data.FontBody == "" {
		data.FontHeading, data.FontBody = ResolveFontFamilies(data.FontPreset)
	}
	if data.AuthMode == "" {
		data.AuthMode = "register"
	}

	var buf bytes.Buffer
	if err := authTmpl.Execute(&buf, data); err != nil {
		return err
	}
	return writeHTMLWithConditionalCache(w, r, buf.Bytes())
}
