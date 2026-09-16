package adminstudio

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"strings"
)

var ErrInvalidForm = errors.New("invalid admin form")

type FormData struct {
	Collection      Collection
	Action          string
	CSRFToken       string
	Values          map[string]string
	RelationOptions map[string][]RelationOption
	Locales         []string
	CurrentLocale   string
	SubmitLabel     string
	HideSubmit      bool
	Nonce           string
}

type RelationOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

var createFormTemplate = template.Must(template.New("admin-create-form").Parse(`{{range .Collection.Fields}}{{$field := .}}{{if and (ne .Kind "relation") (not .ReadOnly)}}{{if eq .Widget "hidden"}}<input type="hidden" name="{{.ID}}" value="{{index $.Values .ID}}">{{else}}{{if eq .Widget "media"}}<div class="media-select-field"><label>{{.Label}}{{if .Required}} *{{end}}<input id="media_{{.ID}}" name="{{.ID}}" value="{{index $.Values .ID}}" placeholder="Medyadan seçin"{{if .Required}} required{{end}}></label><button class="button button-quiet media-picker-open" type="button" data-media-target="media_{{.ID}}">Kütüphaneden seç</button></div>{{else if or (eq .Widget "list") (eq .ID "key_takeaways")}}<div class="studio-form-field full-width item-list-widget" data-item-list data-field-name="{{.ID}}"><span class="studio-form-label">{{.Label}}{{if .Required}} *{{end}}</span><div class="item-list-container"><div class="item-list-rows"></div><button type="button" class="button button-quiet item-list-add-btn">+ Madde ekle</button></div><textarea class="item-list-hidden-input" name="{{.ID}}" hidden>{{index $.Values .ID}}</textarea></div>{{else if or (eq .Widget "richtext") (eq .ID "body") (eq .ID "description")}}<div class="studio-form-field full-width richtext-wrap" data-field-name="{{.ID}}"><span class="studio-form-label">{{.Label}}{{if .Required}} *{{end}}</span><div class="richtext-field" data-richtext><div class="richtext-toolbar ql-toolbar ql-snow"><span class="ql-formats"><select class="ql-header" aria-label="Başlık boyutu"><option value="1">Başlık 1</option><option value="2">Başlık 2</option><option value="3">Başlık 3</option><option value="4">Başlık 4</option><option selected>Normal Metin</option></select></span><span class="ql-formats"><button type="button" class="ql-bold" aria-label="Kalın"></button><button type="button" class="ql-italic" aria-label="İtalik"></button><button type="button" class="ql-underline" aria-label="Altı çizili"></button><button type="button" class="ql-strike" aria-label="Üstü çizili"></button><button type="button" class="ql-code" aria-label="Satır içi kod"></button></span><span class="ql-formats"><button type="button" class="ql-link" aria-label="Bağlantı ekle"></button><button type="button" class="ql-image" aria-label="Görsel ekle"></button><button type="button" class="ql-blockquote" aria-label="Alıntı"></button><button type="button" class="ql-code-block" aria-label="Kod bloğu"></button></span><span class="ql-formats"><button type="button" class="ql-list" value="ordered" aria-label="Numaralı liste"></button><button type="button" class="ql-list" value="bullet" aria-label="Madde listesi"></button></span><span class="ql-formats"><button type="button" class="ql-clean" aria-label="Biçimlendirmeyi temizle"></button></span></div><div class="richtext-editor ql-container ql-snow"></div><textarea class="richtext-value" name="{{.ID}}" hidden>{{index $.Values .ID}}</textarea></div></div>{{else}}<label>{{.Label}}{{if .Required}} *{{end}}{{if or (eq .Kind "status") (gt (len .Options) 0)}}<select name="{{.ID}}"{{if .Required}} required{{end}}>{{range .Options}}<option value="{{.}}"{{if eq (index $.Values $field.ID) .}} selected{{end}}>{{.}}</option>{{end}}</select>{{else if eq .Widget "textarea"}}<textarea name="{{.ID}}"{{if .Required}} required{{end}}>{{index $.Values .ID}}</textarea>{{else}}<input name="{{.ID}}" type="{{if eq .Kind "number"}}number{{else}}text{{end}}" value="{{index $.Values .ID}}"{{if .Required}} required{{end}}>{{end}}</label>{{end}}{{end}}{{end}}{{end}}<input type="hidden" name="_csrf" value="{{.CSRFToken}}">{{if not .HideSubmit}}<button type="submit">{{if .SubmitLabel}}{{.SubmitLabel}}{{else}}Create{{end}}</button>{{end}}{{if .Nonce}}<div id="admin-media-picker-modal" class="admin-media-picker-modal" hidden><div class="admin-media-picker-dialog" role="dialog" aria-modal="true" aria-label="Medya kütüphanesi"><div><p class="eyebrow">MEDYA KÜTÜPHANESİ</p><h2>Görsel seç</h2></div><button class="button button-quiet" type="button" data-media-close>Kapat</button><p>Yalnız hazır görseller listelenir. Yeni görseli Medya alanından yükleyin.</p><div id="admin-media-picker-grid" class="admin-media-picker-grid"></div></div></div><style nonce="{{.Nonce}}">.media-select-field{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:8px;align-items:end}.media-select-field label{min-width:0}.media-select-field .button{padding:10px 12px;white-space:nowrap}.admin-media-picker-modal{position:fixed;inset:0;z-index:200;display:grid;place-items:center;padding:24px;background:rgba(15,23,42,.55);backdrop-filter:blur(6px);animation:modalFadeIn var(--motion-fast) var(--ease-out) forwards}.admin-media-picker-modal[hidden]{display:none!important}.admin-media-picker-dialog{width:min(860px,100%);max-height:min(78vh,760px);overflow:auto;padding:24px;background:var(--surface);border:1px solid var(--line);border-radius:var(--radius-xl);box-shadow:var(--shadow-lg);animation:modalScaleIn var(--motion-normal) var(--ease-emphasized) forwards}.admin-media-picker-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:12px;margin-top:18px}.admin-media-picker-item{display:grid;gap:7px;padding:8px;border:1px solid var(--line);border-radius:var(--radius-md);background:var(--bg-subtle);color:var(--ink);text-align:left;font:inherit;cursor:pointer;transition:var(--transition)}.admin-media-picker-item:hover{border-color:var(--brand);background:var(--brand-subtle);transform:translateY(-1px)}.admin-media-picker-item img{width:100%;aspect-ratio:4/3;object-fit:cover;border-radius:6px;background:var(--surface-hover)}.admin-media-picker-item small{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--muted)}@media(max-width:620px){.media-select-field{grid-template-columns:1fr}.media-select-field .button{width:100%}}</style><script nonce="{{.Nonce}}">(function(){if(window.HydroInitQuill)window.HydroInitQuill();if(window.HydroInitItemLists)window.HydroInitItemLists();var modal=document.getElementById('admin-media-picker-modal'),grid=document.getElementById('admin-media-picker-grid'),target=null,onSelectCallback=null;window.HydroOpenMediaPicker=function(cb){onSelectCallback=cb;target=null;openModal();};function openModal(){if(!modal||!grid)return;modal.hidden=false;grid.innerHTML='<div class="studio-skeleton skeleton-card" style="grid-column:1/-1;"></div>';fetch('/api/media',{headers:{Accept:'application/json'}}).then(function(res){if(!res.ok)throw new Error('library unavailable');return res.json();}).then(function(assets){grid.innerHTML='';assets.filter(function(asset){return asset.status==='ready'&&asset.mime_type.indexOf('image/')===0}).forEach(function(asset){var button=document.createElement('button'),image=document.createElement('img'),name=document.createElement('small');button.type='button';button.className='admin-media-picker-item';button.dataset.assetId=asset.id;image.src='/media/'+encodeURIComponent(asset.id);image.alt=asset.original_filename;name.textContent=asset.original_filename;button.append(image,name);grid.append(button)});if(!grid.childElementCount)grid.textContent='Seçilebilecek hazır görsel yok.'}).catch(function(){grid.textContent='Medya kütüphanesi açılamadı.'});}if(!modal||!grid)return;function close(){modal.hidden=true;target=null;onSelectCallback=null;}document.addEventListener('click',function(event){var opener=event.target.closest('.media-picker-open');if(opener){target=document.getElementById(opener.dataset.mediaTarget);if(!target)return;onSelectCallback=null;openModal();}if(event.target.closest('[data-media-close]'))close();var choice=event.target.closest('.admin-media-picker-item');if(choice){if(onSelectCallback){onSelectCallback(choice.dataset.assetId);close();}else if(target){target.value=choice.dataset.assetId;target.dispatchEvent(new Event('input',{bubbles:true}));close();}}})})();</script>{{end}}`))

// RenderCreateForm renders metadata-driven controls only. The surrounding
// module handler remains responsible for authorization, validation, service
// invocation, audit, and response navigation.
func RenderCreateForm(w io.Writer, data FormData) error {
	if _, err := NewRegistry([]Collection{data.Collection}); err != nil || data.Action == "" || data.Action[0] != '/' || (len(data.Action) > 1 && data.Action[1] == '/') || data.CSRFToken == "" {
		return ErrInvalidForm
	}
	if data.Values == nil {
		data.Values = map[string]string{}
	}
	var rendered bytes.Buffer
	if err := createFormTemplate.Execute(&rendered, data); err != nil {
		return err
	}
	var relations strings.Builder
	for _, field := range data.Collection.Fields {
		if field.Kind != FieldRelation || field.ReadOnly {
			continue
		}
		relations.WriteString(`<label>` + template.HTMLEscapeString(field.Label))
		if field.Required {
			relations.WriteString(` *`)
		}
		relations.WriteString(`<select name="` + template.HTMLEscapeString(field.ID) + `"`)
		if field.Required {
			relations.WriteString(` required`)
		}
		relations.WriteString(`><option value="">Seçin</option>`)
		for _, option := range data.RelationOptions[field.ID] {
			relations.WriteString(`<option value="` + template.HTMLEscapeString(option.ID) + `"`)
			if data.Values[field.ID] == option.ID {
				relations.WriteString(` selected`)
			}
			relations.WriteString(`>` + template.HTMLEscapeString(option.Label) + `</option>`)
		}
		relations.WriteString(`</select></label>`)
	}
	if relations.Len() > 0 {
		rendered.WriteString(relations.String())
	}
	if len(data.Locales) > 0 {
		var locale strings.Builder
		locale.WriteString(`<label>İçerik dili<select name="locale" aria-label="İçerik dili">`)
		for _, item := range data.Locales {
			item = strings.TrimSpace(item)
			if item == "" || len(item) > 16 {
				continue
			}
			locale.WriteString(`<option value="` + template.HTMLEscapeString(item) + `"`)
			if item == data.CurrentLocale {
				locale.WriteString(` selected`)
			}
			locale.WriteString(`>` + template.HTMLEscapeString(item) + `</option>`)
		}
		locale.WriteString(`</select></label>`)
		rendered.WriteString(locale.String())
	}
	richTextLabel := "Zengin metin alanı"
	for _, field := range data.Collection.Fields {
		if field.Widget == "richtext" || field.ID == "body" || field.ID == "description" {
			richTextLabel = field.Label
			break
		}
	}
	// Quill turns this container into a contenteditable textbox at runtime;
	// keep its accessible name on the server-rendered shell as well.
	html := strings.ReplaceAll(rendered.String(),
		`class="richtext-editor ql-container ql-snow"`,
		`class="richtext-editor ql-container ql-snow" role="textbox" aria-label="`+template.HTMLEscapeString(richTextLabel)+`" aria-multiline="true"`)
	_, err := io.WriteString(w, html)
	return err
}
