package transport

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	uicomponents "github.com/ardaozyildirim/hydrogo/internal/ui/components"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) navigationEditor(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	locales := supportedNavigationLocales(h.NavigationLocales)
	locale := canonical.Locale(r.URL.Query().Get("locale"))
	if !containsString(locales, locale) {
		locale = locales[0]
	}
	items, err := h.Service.ListAdminNavigation(r.Context(), locale)
	if err != nil {
		http.Error(w, "Menü bilgileri şu anda açılamıyor.", http.StatusServiceUnavailable)
		return
	}
	var suggestions []site.NavigationSuggestion
	if h.NavigationSuggestions != nil {
		suggestions, _ = h.NavigationSuggestions(r.Context(), locale)
	}
	if ui.AdminStartForRequest(w, r, "Menü düzenleyici", "Bağlantıları sıralayın ve alt menüler oluşturun", "site") != nil {
		return
	}
	_, _ = io.WriteString(w, `<main class="studio-card"><nav aria-label="Menü dili">`)
	for _, activeLocale := range locales {
		_, _ = fmt.Fprintf(w, `<a class="button button-quiet" href="/admin/site/navigation/editor?locale=%s">%s</a>`, url.QueryEscape(activeLocale), template.HTMLEscapeString(strings.ToUpper(activeLocale)))
	}
	_, _ = fmt.Fprintf(w, `</nav><p>Bağlantıları sürükleyerek sıralayın. Üst menü seçerek açılır alt menü oluşturabilirsiniz.</p><form method="post" action="/admin/site/navigation/editor"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="locale" value="%s">`, template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), template.HTMLEscapeString(locale))
	if len(suggestions) > 0 {
		_, _ = io.WriteString(w, `<datalist id="published-page-paths">`)
		for _, suggestion := range suggestions {
			_, _ = fmt.Fprintf(w, `<option value="%s">%s</option>`, template.HTMLEscapeString(suggestion.Href), template.HTMLEscapeString(suggestion.Label))
		}
		_, _ = io.WriteString(w, `</datalist>`)
	}
	_, _ = io.WriteString(w, `<div class="table-wrap"><table><thead><tr><th scope="col">Sıra</th><th scope="col">Bağlantı</th><th scope="col">Üst menü</th><th scope="col">Açılış</th><th scope="col">Durum</th></tr></thead><tbody id="navigation-tree">`)
	for _, item := range items {
		if err := renderNavigationEditorRow(r.Context(), w, item, items); err != nil {
			return
		}
	}
	_, _ = io.WriteString(w, `</tbody></table></div>
<template id="navigation-row-template">
<tr data-navigation-row draggable="true"><td>
<button type="button" class="button button-quiet" data-move-up aria-label="Yukarı taşı">↑</button>
<button type="button" class="button button-quiet" data-move-down aria-label="Aşağı taşı">↓</button>
<input type="hidden" name="id"><input type="hidden" name="position"></td>
<td><label>Bağlantı adı<input name="label" maxlength="256" required></label>
<label>Sayfa yolu<input name="href" maxlength="2048" placeholder="/hakkimizda" list="published-page-paths" required></label></td>
<td><label>Üst menü<select name="parent_id"><option value="">Ana menü</option></select></label></td>
<td><label>Açılış şekli<select name="target"><option value="_self">Aynı sekmede</option><option value="_blank">Yeni sekmede</option></select></label></td>
<td><label>Görünürlük<select name="visible"><option value="true">Yayında</option><option value="false">Gizli</option></select></label>
<button type="button" class="button button-quiet" data-remove-row>Bağlantıyı kaldır</button></td></tr>
</template>
<div class="studio-actions"><button class="button button-quiet" type="button" id="navigation-add">Bağlantı ekle</button><button class="button" type="submit">Menüyü kaydet</button></div></form></main>`)
	_, _ = fmt.Fprint(w, `<script>(()=>{
const tree=document.getElementById('navigation-tree'),tpl=document.getElementById('navigation-row-template');if(!tree||!tpl)return;
const rows=()=>[...tree.querySelectorAll('[data-navigation-row]')];
const refresh=()=>{const opts=rows().map(r=>({id:r.querySelector('[name=id]').value,label:r.querySelector('[name=label]').value||'Yeni bağlantı'}));for(const row of rows()){const s=row.querySelector('[name=parent_id]'),value=s.value,id=row.querySelector('[name=id]').value;s.replaceChildren(new Option('Ana menü',''));opts.filter(x=>x.id!==id).forEach(x=>s.add(new Option(x.label,x.id)));s.value=value}};
const renumber=()=>rows().forEach((r,i)=>r.querySelector('[name=position]').value=String(i));
document.getElementById('navigation-add').addEventListener('click',()=>{const row=tpl.content.firstElementChild.cloneNode(true);row.querySelector('[name=id]').value=globalThis.crypto?.randomUUID?.()||'nav-'+Date.now()+'-'+Math.random().toString(36).slice(2);tree.append(row);refresh();renumber();row.querySelector('[name=label]').focus()});
tree.addEventListener('input',e=>{if(e.target.matches('[name=label]'))refresh()});
tree.addEventListener('click',e=>{const row=e.target.closest('[data-navigation-row]');if(!row)return;if(e.target.closest('[data-remove-row]'))row.remove();if(e.target.closest('[data-move-up]')&&row.previousElementSibling)tree.insertBefore(row,row.previousElementSibling);if(e.target.closest('[data-move-down]')&&row.nextElementSibling)tree.insertBefore(row.nextElementSibling,row);refresh();renumber()});
let dragging=null;tree.addEventListener('dragstart',e=>{dragging=e.target.closest('[data-navigation-row]');if(dragging)e.dataTransfer.setData('text/plain','navigation-row')});tree.addEventListener('dragover',e=>{const target=e.target.closest('[data-navigation-row]');if(!dragging||!target||target===dragging)return;e.preventDefault();const box=target.getBoundingClientRect();tree.insertBefore(dragging,e.clientY<box.top+box.height/2?target:target.nextElementSibling)});tree.addEventListener('dragend',()=>{dragging=null;renumber()});
tree.addEventListener('change',e=>{if(e.target.matches('[name=parent_id]')){const row=e.target.closest('[data-navigation-row]'),parent=rows().find(x=>x.querySelector('[name=id]').value===e.target.value);if(parent&&row&&parent.nextElementSibling!==row)tree.insertBefore(row,parent.nextElementSibling)}renumber()});
refresh();renumber();document.querySelectorAll('#navigation-tree [name=href]').forEach(input=>input.setAttribute('list','published-page-paths'));
})();</script>`)
	_ = ui.AdminEnd(w)
}

func supportedNavigationLocales(configured []string) []string {
	if len(configured) == 0 {
		configured = []string{"tr", "en"}
	}
	locales := make([]string, 0, len(configured))
	for _, raw := range configured {
		locale := canonical.Locale(raw)
		if locale != "" && !containsString(locales, locale) {
			locales = append(locales, locale)
		}
	}
	if len(locales) == 0 {
		return []string{"tr"}
	}
	return locales
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func renderNavigationEditorRow(ctx context.Context, output io.Writer, item site.NavigationItem, items []site.NavigationItem) error {
	e := template.HTMLEscapeString
	if _, err := fmt.Fprintf(output, `<tr data-navigation-row draggable="true"><td><button type="button" class="button button-quiet" data-drag-handle aria-label="Sürükleyerek sırala">↕</button><button type="button" class="button button-quiet" data-move-up aria-label="Yukarı taşı">↑</button><button type="button" class="button button-quiet" data-move-down aria-label="Aşağı taşı">↓</button><input type="hidden" name="id" value="%s"><input type="hidden" name="position" value="%d"></td><td>`, e(item.ID), item.Position); err != nil {
		return err
	}
	for _, field := range []uicomponents.InputProps{
		{ID: "nav-label-" + item.ID, Name: "label", Label: "Bağlantı adı", Value: item.Label, Required: true, HasMaxLength: true, MaxLength: 256},
		{ID: "nav-href-" + item.ID, Name: "href", Label: "Sayfa yolu", Value: item.Href, Required: true, HasMaxLength: true, MaxLength: 2048},
	} {
		if err := uicomponents.Input(field).Render(ctx, output); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(output, `</td><td><label for="nav-parent-`+e(item.ID)+`">Üst menü</label><select id="nav-parent-`+e(item.ID)+`" name="parent_id"><option value="">Ana menü</option>`); err != nil {
		return err
	}
	for _, parent := range items {
		if parent.ID == item.ID {
			continue
		}
		selected := ""
		if item.ParentID == parent.ID {
			selected = ` selected="selected"`
		}
		if _, err := fmt.Fprintf(output, `<option value="%s"%s>%s</option>`, e(parent.ID), selected, e(parent.Label)); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(output, `</select></td>`); err != nil {
		return err
	}
	if err := renderNavigationSelect(output, "nav-target-"+item.ID, "target", "Açılış şekli", item.Target, []string{"_self", "_blank"}, map[string]string{"_self": "Aynı sekmede", "_blank": "Yeni sekmede"}); err != nil {
		return err
	}
	visible := "false"
	if item.Visible {
		visible = "true"
	}
	if err := renderNavigationSelect(output, "nav-visible-"+item.ID, "visible", "Görünürlük", visible, []string{"true", "false"}, map[string]string{"true": "Yayında", "false": "Gizli"}); err != nil {
		return err
	}
	_, err := io.WriteString(output, `<button type="button" class="button button-quiet" data-remove-row>Bağlantıyı kaldır</button></td></tr>`)
	return err
}

func renderNavigationSelect(output io.Writer, id, name, label, value string, options []string, labels map[string]string) error {
	e := template.HTMLEscapeString
	if _, err := fmt.Fprintf(output, `<td><label for="%s">%s</label><select id="%s" name="%s">`, e(id), e(label), e(id), e(name)); err != nil {
		return err
	}
	for _, option := range options {
		selected := ""
		if option == value {
			selected = ` selected="selected"`
		}
		if _, err := fmt.Fprintf(output, `<option value="%s"%s>%s</option>`, e(option), selected, e(labels[option])); err != nil {
			return err
		}
	}
	_, err := io.WriteString(output, `</select></td>`)
	return err
}

func (h Handler) saveNavigationEditor(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Menü kaydedilemedi. Alanları kontrol edip tekrar deneyin.", http.StatusBadRequest)
		return
	}
	locale := canonical.Locale(r.FormValue("locale"))
	if !containsString(supportedNavigationLocales(h.NavigationLocales), locale) {
		http.Error(w, "Menü dili geçersiz.", http.StatusBadRequest)
		return
	}
	ids, labels, hrefs := r.Form["id"], r.Form["label"], r.Form["href"]
	parents, targets, visibility := r.Form["parent_id"], r.Form["target"], r.Form["visible"]
	if len(ids) > 500 || len(ids) != len(labels) || len(ids) != len(hrefs) || len(ids) != len(parents) || len(ids) != len(targets) || len(ids) != len(visibility) {
		http.Error(w, "Menüdeki bağlantı sayısı veya alanlar geçersiz.", http.StatusBadRequest)
		return
	}
	items := make([]site.NavigationItem, len(ids))
	for i := range ids {
		items[i] = site.NavigationItem{ID: ids[i], Locale: locale, Label: labels[i], Href: hrefs[i], ParentID: parents[i], Target: targets[i], Position: i, Visible: visibility[i] == "true"}
		if visibility[i] != "true" && visibility[i] != "false" {
			http.Error(w, "Menüdeki görünürlük seçimi geçersiz.", http.StatusBadRequest)
			return
		}
	}
	if err := h.Service.SaveNavigation(r.Context(), locale, items); err != nil {
		http.Error(w, "Menü kaydedilemedi. Bağlantıların ve üst menü seçimlerinin doğru olduğundan emin olun.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/navigation/editor?locale="+url.QueryEscape(locale)+"&saved=1", http.StatusSeeOther)
}
