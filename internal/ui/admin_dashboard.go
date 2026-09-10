package ui

import (
	"html/template"
	"io"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
)

type AdminDashboardData struct {
	Email       string
	Collections []adminstudio.Collection
}

var adminDashboardTemplate = template.Must(template.New("admin-dashboard").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo Admin</title>
<style>body{margin:0;background:#eef3f9;color:#132238;font-family:system-ui,sans-serif}.wrap{width:min(1080px,calc(100% - 32px));margin:32px auto}.head,.card{background:#fff;border:1px solid #dbe5f3;border-radius:18px;padding:24px;box-shadow:0 12px 36px #18305614}.head{display:flex;justify-content:space-between;gap:16px;align-items:center}.head h1{margin:0}.muted{color:#607089}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(230px,1fr));gap:16px;margin-top:18px}.card h2{margin-top:0}.card a{display:block;color:#0b3da8;font-weight:700;text-decoration:none;padding:8px 0}.badge{display:inline-block;padding:5px 9px;border-radius:999px;background:#e7f0ff;color:#0b3da8;font-size:.8rem;font-weight:700}</style></head>
<body><main class="wrap"><section class="head"><div><span class="badge">ADMIN CONSOLE</span><h1>HydroGo yönetim merkezi</h1><p class="muted">{{.Email}} · içerik, yayın ve commerce araçları</p></div><a href="/public/pages/tr/demo">Siteyi görüntüle</a></section>
<section class="grid">{{range .Collections}}<article class="card"><h2>{{.Label}}</h2><p class="muted">{{.Description}}</p><p class="muted">Alanlar: {{range .Fields}}<span>{{.Label}}</span> {{end}}</p>{{range .Actions}}<a href="{{.Href}}">{{.Label}}</a>{{end}}</article>{{end}}
<article class="card"><h2>API ve sistem</h2><a href="/api/v1/commerce/products">Commerce REST API</a><a href="/public/news/tr/release-notes">Public news API</a><a href="/public/pages/tr/showcase/html">Rendered theme preview</a><p class="muted">GraphQL: POST /api/graphql<br>Health: /healthz · Ready: /readyz</p></article></section></main></body></html>`))

func RenderAdminDashboard(w io.Writer, data AdminDashboardData) error {
	return adminDashboardTemplate.Execute(w, data)
}
