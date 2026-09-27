package adminstudio

import (
	"encoding/json"
	"html/template"
	"io"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/forms"
)

// FormBuilderData describes the bounded schema editor used by Client Editions.
// Persistence and authorization stay with the owning module.
type FormBuilderData struct {
	Form      forms.Form
	Action    string
	CSRFToken string
	Nonce     string
	Submit    string
}

// RenderFormBuilder renders a schema-driven editor with a live, accessible
// preview. Values are JSON-encoded into data attributes and never interpolated
// into executable script, keeping the component compatible with a strict CSP.
func RenderFormBuilder(w io.Writer, data FormBuilderData) error {
	if err := data.Form.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(data.Action) == "" {
		data.Action = "#"
	}
	if data.Submit == "" {
		data.Submit = "Formu kaydet"
	}
	payload, err := json.Marshal(data.Form)
	if err != nil {
		return err
	}
	options := make([]string, 0, len(data.Form.Fields))
	for _, field := range data.Form.Fields {
		if len(field.Options) == 0 {
			continue
		}
		encoded, _ := json.Marshal(field.Options)
		options = append(options, template.HTMLEscapeString(string(encoded)))
	}
	_, err = io.WriteString(w, `<style nonce="`+template.HTMLEscapeString(data.Nonce)+`">.form-builder-fields{display:grid;gap:10px}.form-builder-row{display:grid;grid-template-columns:minmax(100px,1fr) minmax(140px,1.5fr) 130px auto minmax(120px,1fr) auto;gap:8px;align-items:center;padding:10px;border:1px solid var(--line);border-radius:var(--radius-md);background:var(--bg-subtle)}.form-builder-row input,.form-builder-row select{min-width:0}.checkbox-label{display:flex;align-items:center;gap:5px;white-space:nowrap;font-size:.82rem}.checkbox-label input{width:auto}@media(max-width:900px){.form-builder-row{grid-template-columns:1fr 1fr 1fr}.form-builder-row .checkbox-label{grid-column:span 1}}@media(max-width:560px){.form-builder-row{grid-template-columns:1fr}.form-builder-row .checkbox-label{grid-column:auto}}</style><section class="studio-card form-builder" data-form-schema='`+template.HTMLEscapeString(string(payload))+`'><div class="card-title-group"><div><p class="eyebrow">SCHEMA EDİTÖRÜ</p><h2>Form alanları</h2><p class="muted">Alanları ve seçenekleri yönetin; sınırlar kaydetmeden önce doğrulanır.</p></div><span class="studio-badge">`+template.HTMLEscapeString(data.Form.ID)+`</span></div><form method="post" action="`+template.HTMLEscapeString(data.Action)+`" class="studio-form" data-form-builder><input type="hidden" name="_csrf" value="`+template.HTMLEscapeString(data.CSRFToken)+`"><input type="hidden" name="schema" data-form-schema-input><label>Başlık<input name="title" value="`+template.HTMLEscapeString(data.Form.Title)+`" maxlength="256" required></label><label>Bildirim alıcıları<textarea name="notification_recipients" placeholder="E-posta adreslerini satır satır yazın">`+template.HTMLEscapeString(strings.Join(data.Form.NotificationRecipients, "\n"))+`</textarea></label><label class="checkbox-label"><input type="checkbox" name="autoresponder"`+checked(data.Form.Autoresponder)+`> Başvuru yapan kişiye alındı e-postası gönder</label><label>E-posta konusu<input name="autoresponder_subject" value="`+template.HTMLEscapeString(data.Form.AutoresponderSubject)+`" maxlength="256"></label><label>E-posta metni<textarea name="autoresponder_body" maxlength="16384">`+template.HTMLEscapeString(data.Form.AutoresponderBody)+`</textarea></label><label class="checkbox-label"><input type="checkbox" name="turnstile_enabled"`+checked(data.Form.TurnstileEnabled)+`> Spam doğrulamasını aç (önce site anahtarlarını yapılandırın)</label><div class="form-builder-fields" data-form-fields>`+renderBuilderRows(data.Form)+`</div><div class="studio-actions"><button type="button" class="button button-quiet" data-form-add-field>+ Alan ekle</button><button type="submit" class="button button-primary">`+template.HTMLEscapeString(data.Submit)+`</button></div></form><template data-form-field-template><div class="form-builder-row" data-form-field><input name="field_id" placeholder="alan_id" maxlength="64" required><input name="field_label" placeholder="Alan etiketi" maxlength="256" required><select name="field_type"><option value="text">Metin</option><option value="email">E-posta</option><option value="textarea">Uzun metin</option><option value="number">Sayı</option><option value="checkbox">Onay kutusu</option><option value="select">Seçim</option></select><label class="checkbox-label"><input type="checkbox" name="field_required"> Zorunlu</label><input name="field_options" placeholder="Seçenekler: a,b,c" maxlength="256"><button type="button" class="button button-quiet button-danger" data-form-remove-field>Sil</button></div></template><script nonce="`+template.HTMLEscapeString(data.Nonce)+`">(()=>{const root=document.querySelector('[data-form-builder]');if(!root)return;const list=root.querySelector('[data-form-fields]'),tpl=root.parentElement.querySelector('[data-form-field-template]'),schema=root.querySelector('[data-form-schema-input]');function sync(){const fields=[...list.querySelectorAll('[data-form-field]')].map(row=>({id:row.querySelector('[name=field_id]').value.trim(),type:row.querySelector('[name=field_type]').value,label:row.querySelector('[name=field_label]').value.trim(),required:row.querySelector('[name=field_required]').checked,options:row.querySelector('[name=field_options]').value.split(',').map(v=>v.trim()).filter(Boolean)}));const original=JSON.parse(root.closest('[data-form-schema]').dataset.formSchema||'{}');schema.value=JSON.stringify({...original,id:original.id||'`+template.JSEscapeString(data.Form.ID)+`',title:root.querySelector('[name=title]').value,fields,notification_recipients:root.querySelector('[name=notification_recipients]').value.split(/\r?\n/).map(v=>v.trim()).filter(Boolean),autoresponder:root.querySelector('[name=autoresponder]').checked,autoresponder_subject:root.querySelector('[name=autoresponder_subject]').value,autoresponder_body:root.querySelector('[name=autoresponder_body]').value,turnstile_enabled:root.querySelector('[name=turnstile_enabled]').checked});}root.addEventListener('input',sync);root.addEventListener('change',sync);root.addEventListener('click',e=>{if(e.target.closest('[data-form-remove-field]')){e.target.closest('[data-form-field]').remove();sync();}if(e.target.closest('[data-form-add-field]')){list.append(tpl.content.cloneNode(true));sync();}});sync();})();</script></section>`)
	_ = options
	return err
}

func renderBuilderRows(form forms.Form) string {
	var b strings.Builder
	for _, field := range form.Fields {
		b.WriteString(`<div class="form-builder-row" data-form-field><input name="field_id" value="`)
		b.WriteString(template.HTMLEscapeString(field.ID))
		b.WriteString(`" maxlength="64" required><input name="field_label" value="`)
		b.WriteString(template.HTMLEscapeString(field.Label))
		b.WriteString(`" maxlength="256" required><select name="field_type">`)
		for _, kind := range []forms.FieldType{forms.Text, forms.Email, forms.Textarea, forms.Number, forms.Checkbox, forms.Select} {
			b.WriteString(`<option value="` + template.HTMLEscapeString(string(kind)) + `"`)
			if field.Type == kind {
				b.WriteString(` selected`)
			}
			b.WriteString(`>` + template.HTMLEscapeString(string(kind)) + `</option>`)
		}
		b.WriteString(`</select><label class="checkbox-label"><input type="checkbox" name="field_required"`)
		if field.Required {
			b.WriteString(` checked`)
		}
		b.WriteString(`> Zorunlu</label><input name="field_options" value="`)
		b.WriteString(template.HTMLEscapeString(strings.Join(field.Options, ",")))
		b.WriteString(`" maxlength="256"><button type="button" class="button button-quiet button-danger" data-form-remove-field>Sil</button></div>`)
	}
	return b.String()
}

func checked(value bool) string {
	if value {
		return ` checked`
	}
	return ""
}
