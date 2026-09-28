package ui

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type NotificationItem struct {
	ID, Title, Message, TargetPath, Created string
	Read                                    bool
}

const AdminBusinessCSS = `.business-home{max-width:980px;margin:24px auto}.business-home h2{font-size:1.6rem}.business-home>p{color:var(--muted)}.business-home nav{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px}.business-home nav a{display:grid;gap:8px;text-decoration:none;color:var(--ink)}.business-home nav a span{font-size:.82rem;color:var(--muted)}.studio-support-card{display:grid;gap:5px;padding:10px 12px;border-top:1px solid var(--line);font-size:.72rem}.studio-support-card a{color:var(--brand-light);overflow-wrap:anywhere}.studio-support-sla{width:max-content;padding:2px 6px;border-radius:999px;background:var(--green-subtle);color:var(--green)}@media(max-width:700px){.business-home nav{grid-template-columns:1fr}}`

func renderAdminSupport(config adminstudio.SupportConfig) string {
	var out strings.Builder
	out.WriteString(`<section class="studio-support-card" aria-label="Destek"><strong>` + template.HTMLEscapeString(config.AgencyName) + `</strong>`)
	if config.SLABadge != "" {
		out.WriteString(`<span class="studio-support-sla">` + template.HTMLEscapeString(config.SLABadge) + `</span>`)
	}
	if config.Email != "" {
		escaped := template.HTMLEscapeString(config.Email)
		out.WriteString(`<a href="mailto:` + escaped + `">` + escaped + `</a>`)
	}
	if config.Phone != "" {
		escaped := template.HTMLEscapeString(config.Phone)
		out.WriteString(`<a href="tel:` + escaped + `">` + escaped + `</a>`)
	}
	if config.WhatsAppURL != "" {
		out.WriteString(`<a href="` + template.HTMLEscapeString(config.WhatsAppURL) + `" target="_blank" rel="noopener noreferrer">WhatsApp desteği</a>`)
	}
	if config.GuideURL != "" {
		out.WriteString(`<a href="` + template.HTMLEscapeString(config.GuideURL) + `" target="_blank" rel="noopener noreferrer">Video rehberi</a>`)
	}
	out.WriteString(`</section>`)
	return out.String()
}

const AdminUXHotkeysJS = `document.addEventListener('keydown',e=>{const tag=document.activeElement?.tagName,typing=/INPUT|TEXTAREA|SELECT/.test(tag)||document.activeElement?.isContentEditable;if((e.metaKey||e.ctrlKey)&&e.key.toLowerCase()==='s'){const form=document.activeElement?.closest('form')||document.querySelector('main form');if(form){e.preventDefault();form.requestSubmit()}return}if(typing||e.metaKey||e.ctrlKey||e.altKey)return;if(e.key.toLowerCase()==='n'){const link=document.querySelector('.heading-actions a');if(link)location.assign(link.href)}if(e.key.toLowerCase()==='j'||e.key.toLowerCase()==='k'){const rows=[...document.querySelectorAll('tbody tr')];if(!rows.length)return;const active=rows.indexOf(document.activeElement?.closest('tr'));const next=Math.max(0,Math.min(rows.length-1,active+(e.key.toLowerCase()==='j'?1:-1)));rows[next].tabIndex=0;rows[next].focus()}});`

const AdminUXCSS = `body[data-studio-role="editor"] [data-admin-only]{display:none!important}.studio-ux-trigger{position:relative;display:grid;place-items:center;width:34px;height:34px;border:1px solid var(--line);border-radius:var(--radius-md);background:var(--surface);color:var(--ink);cursor:pointer}.studio-ux-trigger{position:relative;display:grid;place-items:center;width:34px;height:34px;border:1px solid var(--line);border-radius:var(--radius-md);background:var(--surface);color:var(--ink);cursor:pointer}.admin-ux-dialog{position:fixed;inset:0;z-index:10000;display:grid;place-items:center;padding:16px;background:rgba(2,6,23,.62)}.admin-ux-dialog[hidden]{display:none}.admin-ux-panel{position:relative;width:min(520px,100%);max-height:90vh;overflow:auto;padding:24px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);color:var(--ink);box-shadow:var(--shadow-lg)}.admin-ux-panel h2{margin:0 32px 16px 0;font-size:1.2rem}.admin-ux-panel form{display:grid;gap:12px}.admin-ux-panel label{display:grid;gap:5px}.admin-ux-panel input:not([type=checkbox]){width:100%;min-height:40px;padding:8px 10px;border:1px solid var(--line);border-radius:var(--radius-sm);background:var(--bg-subtle);color:var(--ink)}.admin-ux-panel meter{width:100%}.admin-ux-close{position:absolute;right:16px;top:12px;border:0;background:transparent;color:var(--muted);font-size:24px;cursor:pointer}.admin-ux-panel dl{display:grid;grid-template-columns:max-content 1fr;gap:10px 18px}.admin-ux-panel dt{font-weight:700}.admin-ux-panel dd{margin:0;color:var(--muted)}.admin-ux-panel nav{display:flex;gap:12px;flex-wrap:wrap;margin:16px 0}.studio-bulk-floating{position:fixed;z-index:9000;left:50%;bottom:18px;transform:translateX(-50%);display:flex;align-items:center;gap:10px;padding:10px 14px;border:1px solid var(--line);border-radius:var(--radius-lg);background:var(--surface);box-shadow:var(--shadow-lg)}.studio-bulk-floating[hidden]{display:none}.studio-split-editor{display:grid;grid-template-columns:1fr 1fr;gap:16px}.translation-source{white-space:pre-wrap;max-height:70vh;overflow:auto;padding:14px;border:1px solid var(--line);border-radius:var(--radius-sm)}.revision-diff-fields{display:grid;gap:10px}.revision-diff-fields article{padding:10px;border:1px solid var(--line);border-radius:var(--radius-sm)}.revision-removed,.revision-added{padding:8px;white-space:pre-wrap;overflow-wrap:anywhere}.revision-removed{background:var(--red-subtle);color:var(--red);text-decoration:line-through}.revision-added{background:var(--green-subtle);color:var(--green)}@media(max-width:760px){.studio-split-editor{grid-template-columns:1fr}}`

const AdminUXJS = `(function(){const qs=(s,r=document)=>r.querySelector(s),qsa=(s,r=document)=>[...r.querySelectorAll(s)],dialogs=[...qsa('.admin-ux-dialog')];function open(id){const d=document.getElementById(id);if(d){d.hidden=false;d.querySelector('button,input')?.focus()}}function close(d){if(d)d.hidden=true}document.addEventListener('click',e=>{if(e.target.closest('[data-shortcuts-open]'))open('admin-shortcuts-dialog');if(e.target.closest('[data-password-open]'))open('admin-password-dialog');if(e.target.closest('[data-ux-close]')||e.target.classList.contains('admin-ux-dialog'))close(e.target.closest('.admin-ux-dialog')||e.target);if(e.target.closest('[data-onboarding-dismiss]')){localStorage.setItem('hydro-onboarding-dismissed','1');close(e.target.closest('.admin-ux-dialog'))}});document.addEventListener('keydown',e=>{const typing=/INPUT|TEXTAREA|SELECT/.test(document.activeElement?.tagName)||document.activeElement?.isContentEditable;if(e.key==='Escape'){dialogs.forEach(d=>{if(!d.hidden)close(d)});return}if((e.key==='?'||(e.key==='/'&&(e.metaKey||e.ctrlKey)))&&!typing){e.preventDefault();open('admin-shortcuts-dialog')}});document.addEventListener('submit',e=>{const form=e.target.closest('[data-confirm-action]');if(form&&!confirm(form.dataset.confirmAction))e.preventDefault()});const source=qs('[data-translation-source]'),target=qs('[data-translation-target]');if(source&&target){let syncing=false;for(const [a,b] of [[source,target],[target,source]])a.addEventListener('scroll',()=>{if(syncing)return;syncing=true;b.scrollTop=a.scrollTop/(a.scrollHeight-a.clientHeight||1)*(b.scrollHeight-b.clientHeight);requestAnimationFrame(()=>syncing=false)})}const form=qs('[data-quick-password-form]');if(form){const csrf=decodeURIComponent((document.cookie.match(/(?:^|; )hydro_csrf=([^;]+)/)||[])[1]||'');qs('[data-ux-csrf]',form).value=csrf;const pass=qs('[data-password-strength]',form),meter=qs('[data-password-meter]',form);pass.addEventListener('input',()=>{const v=pass.value;meter.value=Math.min(4,(v.length>=12?1:0)+(/[A-Z]/.test(v)?1:0)+(/[0-9]/.test(v)?1:0)+(/[^A-Za-z0-9]/.test(v)?1:0))});form.addEventListener('submit',async e=>{e.preventDefault();const status=qs('[data-quick-password-status]',form);if(form.elements.new_password.value!==form.elements.confirm_password.value){status.textContent='Yeni parolalar eşleşmiyor.';return}status.textContent='Parolan güncelleniyor…';try{const response=await fetch('/auth/me/credentials',{method:'POST',body:new FormData(form),headers:{'X-CSRF-Token':csrf}});if(!response.ok)throw new Error();if(form.elements.invalidate_other_sessions.checked){status.textContent='Parolan güncellendi; diğer cihazlardaki oturumlar kapatıldı.';form.reset();setTimeout(()=>close(qs('#admin-password-dialog')),1300)}else{status.textContent='Parolan güncellendi. Yeniden giriş yapman gerekiyor.';setTimeout(()=>location.assign('/login'),700)}}catch(_){status.textContent='Parola güncellenemedi. Bilgilerini kontrol edip tekrar dene.'}})}const boxes=()=>qsa('[data-bulk-select]');let last=null;function update(){const selected=boxes().filter(b=>b.checked);let bar=qs('[data-bulk-floating]');if(!bar&&selected.length){bar=document.createElement('div');bar.className='studio-bulk-floating';bar.dataset.bulkFloating='';bar.setAttribute('role','toolbar');bar.setAttribute('aria-label','Toplu işlemler');bar.innerHTML='<strong data-bulk-count></strong><button type="button" data-bulk-run>Uygula</button><button type="button" data-bulk-clear>Seçimi kaldır</button>';document.body.append(bar)}if(bar){bar.hidden=!selected.length;qs('[data-bulk-count]',bar).textContent=selected.length+' kayıt seçildi'}}document.addEventListener('click',e=>{const box=e.target.closest('[data-bulk-select]');if(box&&e.shiftKey&&last){const all=boxes(),a=all.indexOf(last),b=all.indexOf(box);all.slice(Math.min(a,b),Math.max(a,b)+1).forEach(x=>x.checked=box.checked)}if(box)last=box;if(e.target.closest('[data-bulk-select-all]'))boxes().forEach(x=>x.checked=e.target.checked);if(e.target.closest('[data-bulk-clear]'))boxes().forEach(x=>x.checked=false);if(e.target.closest('[data-bulk-run]')){const action=qs('[data-bulk-action]'),selected=boxes().filter(x=>x.checked);if(!action||!selected.length||!confirm('Bu işlem seçili '+selected.length+' kayıt için uygulansın mı?'))return;const f=document.createElement('form');f.method='post';f.action=action.getAttribute('href');const csrf=decodeURIComponent((document.cookie.match(/(?:^|; )hydro_csrf=([^;]+)/)||[])[1]||'');[['action',action.dataset.bulkAction],['_csrf',csrf],...selected.map(x=>['ids',x.value])].forEach(([name,value])=>{const i=document.createElement('input');i.type='hidden';i.name=name;i.value=value;f.append(i)});document.body.append(f);f.submit()}if(e.target.closest('[data-bulk-action]')){e.preventDefault();qs('[data-bulk-run]')?.click()}if(e.target.closest('[data-notifications-open]'))location.assign('/admin/notifications')});document.addEventListener('change',e=>{if(e.target.matches('[data-bulk-select],[data-bulk-select-all]'))update()});if(!localStorage.getItem('hydro-onboarding-dismissed')&&!sessionStorage.getItem('hydro-onboarding-seen')){sessionStorage.setItem('hydro-onboarding-seen','1');setTimeout(()=>open('admin-onboarding-dialog'),500)}})();`

func RenderBusinessOwnerDashboard(w http.ResponseWriter, r *http.Request, enabled map[string]bool) error {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleEditor || !user.Active {
		return errors.New("business dashboard is unavailable")
	}
	if err := AdminStartForRequest(w, r, "Hoş geldiniz", "", security.NonceFromContext(r.Context()), "dashboard"); err != nil {
		return err
	}
	if err := BusinessOwnerDashboard(enabled).Render(r.Context(), w); err != nil {
		return err
	}
	return AdminEnd(w)
}
