package ui

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

type NotificationItem struct {
	ID, Title, Message, TargetPath, Created string
	Read                                    bool
}

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
		out.WriteString(`<a href="/admin/guide">Yardım ve video rehberi</a>`)
	}
	out.WriteString(`</section>`)
	return out.String()
}

const AdminUXHotkeysJS = `document.addEventListener('keydown',e=>{const tag=document.activeElement?.tagName,typing=/INPUT|TEXTAREA|SELECT/.test(tag)||document.activeElement?.isContentEditable;if((e.metaKey||e.ctrlKey)&&e.key.toLowerCase()==='s'){const form=document.activeElement?.closest('form')||document.querySelector('main form');if(form){e.preventDefault();form.requestSubmit()}return}if(typing||e.metaKey||e.ctrlKey||e.altKey)return;if(e.key.toLowerCase()==='n'){const link=document.querySelector('.heading-actions a');if(link)location.assign(link.href)}if(e.key.toLowerCase()==='j'||e.key.toLowerCase()==='k'){const rows=[...document.querySelectorAll('tbody tr')];if(!rows.length)return;const active=rows.indexOf(document.activeElement?.closest('tr'));const next=Math.max(0,Math.min(rows.length-1,active+(e.key.toLowerCase()==='j'?1:-1)));rows[next].tabIndex=0;rows[next].focus()}});`

const AdminNotificationJS = `fetch('/admin/notifications/unread',{credentials:'same-origin',headers:{'Accept':'application/json'}}).then(r=>r.ok?r.json():null).then(data=>{const badge=document.querySelector('[data-notification-count]');if(!badge||!data)return;const count=Number(data.unread)||0;badge.textContent=count>99?'99+':String(count);badge.hidden=count===0;const button=badge.closest('button');if(button)button.setAttribute('aria-label',count+' okunmamış bildirim')}).catch(()=>{});`

const AdminStaffNoticeJS = `(()=>{const key='hydro-staff-notice-dismissed:';const apply=()=>document.querySelectorAll('[data-staff-notice]').forEach(notice=>{try{notice.hidden=localStorage.getItem(key+notice.dataset.staffNotice)==='1'}catch{}});if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',apply,{once:true});else apply();document.addEventListener('click',event=>{const button=event.target.closest('[data-staff-notice-dismiss]');if(!button)return;const notice=button.closest('[data-staff-notice]');if(!notice)return;try{localStorage.setItem(key+notice.dataset.staffNotice,'1')}catch{}notice.hidden=true});document.addEventListener('hydro:admin-navigated',apply)})();`

const AdminPageWeightJS = `(()=>{const panel=()=>document.querySelector('[data-page-weight]');let timer=0,revision=0,observed=null,observer=null;const format=value=>value>=1048576?(value/1048576).toFixed(2)+' MB':Math.max(0,Math.round(value/1024))+' KB';const bounded=async(values,limit,fn)=>{const result=new Array(values.length);let next=0;await Promise.all(Array.from({length:Math.min(limit,values.length)},async()=>{for(;;){const index=next++;if(index>=values.length)return;result[index]=await fn(values[index])}}));return result};const size=async value=>{try{const url=new URL(value,location.href);if(url.origin!==location.origin)return 0;const controller=new AbortController(),timeout=setTimeout(()=>controller.abort(),1500);try{const response=await fetch(url,{method:'HEAD',credentials:'same-origin',signal:controller.signal,cache:'force-cache'});return response.ok?Math.max(0,Number(response.headers.get('content-length'))||0):0}finally{clearTimeout(timeout)}}catch{return 0}};const run=async()=>{const view=panel();if(!view)return;const editor=document.querySelector('.richtext-editor .ql-editor,.richtext-editor[contenteditable="true"],textarea[name="body"]');if(!editor){view.hidden=true;return}const sideCol=document.querySelector('.editor-side-col');if(sideCol&&view.parentElement!==sideCol){sideCol.appendChild(view);view.classList.add('page-weight-docked')}if(!view.closest('.editor-side-col')){view.hidden=true;return}view.hidden=false;const current=++revision,html=editor.value??editor.innerHTML??editor.textContent??'',parsed=new DOMParser().parseFromString(html,'text/html'),assets=[...parsed.querySelectorAll('img[src],video[src],audio[src],source[src],script[src]')].slice(0,24).map(asset=>asset.getAttribute('src')).filter(Boolean),styles=[...document.querySelectorAll('link[rel="stylesheet"][href]')].slice(0,12).map(link=>link.href),htmlBytes=new TextEncoder().encode(html).length+8192;const [assetBytes,cssBytes]=await Promise.all([bounded(assets,4,size),bounded(styles,3,size)]);if(current!==revision)return;const assetTotal=assetBytes.reduce((a,b)=>a+b,0),cssTotal=cssBytes.reduce((a,b)=>a+b,0),total=htmlBytes+assetTotal+cssTotal;view.querySelector('[data-page-weight-total]').textContent=format(total);view.querySelector('[data-page-weight-html]').textContent=format(htmlBytes);view.querySelector('[data-page-weight-css]').textContent=format(cssTotal);view.querySelector('[data-page-weight-assets]').textContent=format(assetTotal);const state=view.querySelector('[data-page-weight-state]');state.textContent=total>3*1048576?'Sayfa ağır. Büyük medya dosyalarını küçültmeyi deneyin.':total>1048576?'Sayfa boyutu orta düzeyde.':'Sayfa boyutu iyi görünüyor.';view.dataset.weightState=total>3*1048576?'heavy':total>1048576?'moderate':'fast';const list=view.querySelector('[data-page-weight-large-assets]');list.replaceChildren();assets.forEach((source,index)=>{if(assetBytes[index]<=1048576)return;const assetURL=new URL(source,location.href),parts=assetURL.pathname.split('/').filter(Boolean),assetID=parts[0]==='media'?parts[1]:'';if(!assetID)return;const item=document.createElement('li'),link=document.createElement('a');link.href='/admin/media?asset='+encodeURIComponent(assetID)+'#media-'+encodeURIComponent(assetID);link.textContent='Büyük medya dosyasını aç ('+format(assetBytes[index])+')';link.className='underline';item.append(link);list.append(item)});};const schedule=()=>{clearTimeout(timer);timer=setTimeout(run,300)};const observe=()=>{const editor=document.querySelector('.richtext-editor .ql-editor,.richtext-editor[contenteditable="true"],textarea[name="body"]');if(editor===observed)return;if(observer)observer.disconnect();observed=editor;observer=null;if(editor&&editor.nodeType===1&&editor.matches('.ql-editor,[contenteditable="true"]')){observer=new MutationObserver(schedule);observer.observe(editor,{childList:true,subtree:true,attributes:true,characterData:true})}};document.addEventListener('input',event=>{if(event.target.closest('.richtext-editor,textarea[name="body"]'))schedule()});document.addEventListener('change',event=>{if(event.target.closest('.richtext-editor,textarea[name="body"]'))schedule()});document.addEventListener('hydro:admin-navigated',()=>{observe();schedule()});if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',()=>{observe();schedule()},{once:true});else{observe();schedule()}})();`

const AdminDevicePreviewJS = `(()=>{const getDialog=()=>document.querySelector('[data-device-preview-dialog]');document.addEventListener('click',async event=>{const dialog=getDialog();const open=event.target.closest('[data-device-preview-open]');if(open){if(!dialog)return;if(!dialog.open)dialog.showModal();const status=dialog.querySelector('[data-device-preview-status]');open.disabled=true;if(status)status.textContent='Önizleme hazırlanıyor…';try{const response=await fetch(open.dataset.previewEndpoint,{method:'POST',credentials:'same-origin',headers:{'Accept':'application/json','X-CSRF-Token':open.dataset.previewCsrf||''}});if(!response.ok)throw new Error();const result=await response.json(),parsed=new URL(result.url,location.href);const allowedPaths=['/preview/pages/','/public/news/','/public/pages/'];if(!allowedPaths.some(p=>parsed.pathname.startsWith(p)))throw new Error();const frame=dialog.querySelector('[data-device-preview-frame]');const target=new URL(parsed.pathname+parsed.search,location.href);frame.src=target.href;if(status)status.textContent='Güncel taslak · betikler devre dışı';}catch(_){const status=dialog?.querySelector('[data-device-preview-status]');if(status)status.textContent='Önizleme açılamadı. Taslağı kaydedip yeniden deneyin.';}finally{open.disabled=false}return}if(!dialog)return;const size=event.target.closest('[data-device-preview-size]');if(size&&dialog.contains(size)){dialog.dataset.deviceSize=size.dataset.devicePreviewSize;dialog.querySelectorAll('[data-device-preview-size]').forEach(button=>button.setAttribute('aria-pressed',String(button===size)));return}if(event.target.closest('[data-device-preview-close]'))dialog.close()});document.addEventListener('close',event=>{const dialog=event.target.closest('[data-device-preview-dialog]');if(!dialog)return;const frame=dialog.querySelector('[data-device-preview-frame]');const status=dialog.querySelector('[data-device-preview-status]');if(frame)frame.src='about:blank';if(status)status.textContent='Güncel taslağı yüklemek için önizle butonuna basın.';},{capture:true})})();`

const AdminRealtimeChimeJS = `(()=>{if(!location.pathname.startsWith('/admin'))return;const key='hydro-inquiry-sound-v1',preference=document.querySelector('[data-sound-alert-preference]');const enabled=()=>{try{return localStorage.getItem(key)==='1'}catch(_){return false}};let audio=null,source=null,errors=0;const unlock=()=>{if(!enabled())return;try{const Audio=window.AudioContext||window.webkitAudioContext;if(!Audio)return;if(!audio)audio=new Audio();if(audio.state==='suspended')audio.resume().catch(()=>{})}catch(_){}};const play=()=>{if(!enabled())return;unlock();if(!audio||audio.state!=='running')return;const start=audio.currentTime;[880,1174].forEach((frequency,index)=>{const oscillator=audio.createOscillator(),gain=audio.createGain(),time=start+index*.12;oscillator.type='sine';oscillator.frequency.setValueAtTime(frequency,time);gain.gain.setValueAtTime(.0001,time);gain.gain.exponentialRampToValueAtTime(.08,time+.02);gain.gain.exponentialRampToValueAtTime(.0001,time+.22);oscillator.connect(gain);gain.connect(audio.destination);oscillator.start(time);oscillator.stop(time+.23)})};if(preference){preference.checked=enabled();preference.addEventListener('change',()=>{try{localStorage.setItem(key,preference.checked?'1':'0')}catch(_){}if(preference.checked)unlock()})}if(enabled()){document.addEventListener('pointerdown',unlock,{once:true});document.addEventListener('keydown',unlock,{once:true});try{source=new EventSource('/api/v1/realtime/subscriptions?topics=forms%3Asubmission_created');source.addEventListener('forms:submission_created',play);source.onopen=()=>{errors=0};source.onerror=()=>{if(++errors>=3||source.readyState===EventSource.CLOSED)source.close()}}catch(_){}}})();`

const AdminTourJS = `(function(){
	const dismissedKey = 'hydro-onboarding-dismissed';

	function initTour() {
		const role = document.body?.dataset.studioRole || 'admin';
		const seenKey = 'hydro-tour-seen:' + role;
		const tour = document.querySelector('[data-onboarding-tour]');
		if (!tour) return;
		const dialog = tour.closest('.admin-ux-dialog');
		if (!dialog) return;

		const steps = [...tour.querySelectorAll('[data-tour-step]')];
		if (!steps.length) return;

		const prev = tour.querySelector('[data-tour-prev]');
		const next = tour.querySelector('[data-tour-next]');
		const counter = tour.querySelector('[data-tour-counter]');
		const dots = [...tour.querySelectorAll('[data-tour-dots] .onboarding-dot')];
		let index = 0;

		const titles = [
			'Studio’ya Hoş Geldiniz',
			'İçerik & Yayın Yönetimi',
			'Geliştirici & Sistem Araçları',
			'Kısayollar & Başarı'
		];

		function show(i) {
			index = Math.max(0, Math.min(steps.length - 1, i));
			steps.forEach((step, idx) => {
				step.hidden = idx !== index;
			});
			dots.forEach((dot, idx) => {
				dot.classList.toggle('is-active', idx === index);
			});
			if (prev) prev.disabled = index === 0;
			if (next) {
				next.textContent = index === steps.length - 1 ? 'Rehberi Tamamla ✓' : 'İleri →';
				next.classList.toggle('button-success', index === steps.length - 1);
				next.classList.toggle('button-primary', index !== steps.length - 1);
			}
			if (counter) counter.textContent = 'Adım ' + (index + 1) + ' / ' + steps.length;
			const titleEl = tour.querySelector('[data-tour-title]');
			if (titleEl && titles[index]) titleEl.textContent = titles[index];
		}

		function dismiss() {
			try {
				localStorage.setItem(dismissedKey, '1');
				localStorage.setItem(dismissedKey + ':' + role, '1');
			} catch (_) {}
			dialog.hidden = true;
		}

		tour.addEventListener('click', e => {
			if (e.target.closest('[data-tour-next]')) {
				if (index < steps.length - 1) {
					show(index + 1);
				} else {
					dismiss();
				}
				return;
			}
			if (e.target.closest('[data-tour-prev]') && index > 0) {
				show(index - 1);
				return;
			}
			if (e.target.closest('[data-onboarding-dismiss]')) {
				dismiss();
				return;
			}
		});

		dialog.addEventListener('click', e => {
			if (e.target === dialog || e.target.closest('[data-ux-close]')) {
				dismiss();
			}
		});

		show(0);

		try {
			const isDismissed = localStorage.getItem(dismissedKey) === '1' || localStorage.getItem(dismissedKey + ':' + role) === '1';
			if (isDismissed) {
				dialog.hidden = true;
			} else if (!sessionStorage.getItem(seenKey)) {
				sessionStorage.setItem(seenKey, '1');
				setTimeout(() => { dialog.hidden = false; }, 400);
			}
		} catch (_) {
			dialog.hidden = true;
		}
	}

	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', initTour, { once: true });
	} else {
		initTour();
	}
	document.addEventListener('hydro:admin-navigated', initTour);
})();`

const AdminGuideJS = `document.addEventListener('click',e=>{const tab=e.target.closest('[data-guide-tab]');if(!tab)return;const root=tab.closest('[data-guide-center]');if(!root)return;root.querySelectorAll('[data-guide-tab]').forEach(button=>button.setAttribute('aria-selected',String(button===tab)));root.querySelectorAll('[data-guide-panel]').forEach(panel=>panel.hidden=panel.dataset.guidePanel!==tab.dataset.guideTab)});`

const AdminUXJS = `(function(){const qs=(s,r=document)=>r.querySelector(s),qsa=(s,r=document)=>[...r.querySelectorAll(s)],dialogs=[...qsa('.admin-ux-dialog')];function open(id){const d=document.getElementById(id);if(d){d.hidden=false;d.querySelector('button,input')?.focus()}}function close(d){if(d)d.hidden=true}document.addEventListener('click',e=>{if(e.target.closest('[data-shortcuts-open]'))open('admin-shortcuts-dialog');if(e.target.closest('[data-ux-close]')||e.target.classList.contains('admin-ux-dialog'))close(e.target.closest('.admin-ux-dialog')||e.target);if(e.target.closest('[data-notifications-open]')){if(window.studioNavigate)window.studioNavigate('/admin/notifications',true);else location.assign('/admin/notifications');}});document.addEventListener('keydown',e=>{const typing=/INPUT|TEXTAREA|SELECT/.test(document.activeElement?.tagName)||document.activeElement?.isContentEditable;if(e.key==='Escape'){dialogs.forEach(d=>{if(!d.hidden)close(d)});return}if((e.key==='?'||(e.key==='/'&&(e.metaKey||e.ctrlKey)))&&!typing){e.preventDefault();open('admin-shortcuts-dialog')}});document.addEventListener('submit',e=>{const form=e.target.closest('[data-confirm-action]');if(form&&!confirm(form.dataset.confirmAction))e.preventDefault()});const source=qs('[data-translation-source]'),target=qs('[data-translation-target]');if(source&&target){let syncing=false;for(const [a,b] of [[source,target],[target,source]])a.addEventListener('scroll',()=>{if(syncing)return;syncing=true;b.scrollTop=a.scrollTop/(a.scrollHeight-a.clientHeight||1)*(b.scrollHeight-b.clientHeight);requestAnimationFrame(()=>syncing=false)})}const boxes=()=>qsa('[data-bulk-select]');let last=null;function update(){const selected=boxes().filter(b=>b.checked);let bar=qs('[data-bulk-floating]');if(!bar&&selected.length){bar=document.createElement('div');bar.className='studio-bulk-floating';bar.dataset.bulkFloating='';bar.setAttribute('role','toolbar');bar.setAttribute('aria-label','Toplu işlemler');bar.innerHTML='<strong data-bulk-count></strong><button type="button" data-bulk-run>Uygula</button><button type="button" data-bulk-clear>Seçimi kaldır</button>';document.body.append(bar)}if(bar){bar.hidden=!selected.length;qs('[data-bulk-count]',bar).textContent=selected.length+' kayıt seçildi'}}document.addEventListener('click',e=>{const box=e.target.closest('[data-bulk-select]');if(box&&e.shiftKey&&last){const all=boxes(),a=all.indexOf(last),b=all.indexOf(box);all.slice(Math.min(a,b),Math.max(a,b)+1).forEach(x=>x.checked=box.checked)}if(box)last=box;if(e.target.closest('[data-bulk-select-all]'))boxes().forEach(x=>x.checked=e.target.checked);if(e.target.closest('[data-bulk-clear]'))boxes().forEach(x=>x.checked=false);if(e.target.closest('[data-bulk-run]')){const action=qs('[data-bulk-action]'),selected=boxes().filter(x=>x.checked);if(!action||!selected.length||!confirm('Bu işlem seçili '+selected.length+' kayıt için uygulansın mı?'))return;const f=document.createElement('form');f.method='post';f.action=action.getAttribute('href');const csrf=decodeURIComponent((document.cookie.match(/(?:^|; )hydro_csrf=([^;]+)/)||[])[1]||'');[['action',action.dataset.bulkAction],['_csrf',csrf],...selected.map(x=>['ids',x.value])].forEach(([name,value])=>{const i=document.createElement('input');i.type='hidden';i.name=name;i.value=value;f.append(i)});document.body.append(f);f.submit()}if(e.target.closest('[data-bulk-action]')){e.preventDefault();qs('[data-bulk-run]')?.click()}});document.addEventListener('change',e=>{if(e.target.matches('[data-bulk-select],[data-bulk-select-all]'))update()});})();`

const AdminResetLinkJS = `document.addEventListener('click',async event=>{const button=event.target.closest('[data-password-reset-link]');if(!button)return;const status=button.parentElement.querySelector('[data-password-reset-status]');const cookie=document.cookie.split('; ').find(value=>value.startsWith('hydro_csrf='));const csrf=cookie?decodeURIComponent(cookie.slice('hydro_csrf='.length)):'';button.disabled=true;if(status)status.textContent='Generating a secure link…';try{const endpoint=button.dataset.passwordResetLink;if(!endpoint.startsWith('/auth/users/')||!endpoint.endsWith('/password-reset-link'))throw new Error('endpoint');const response=await fetch(endpoint,{method:'POST',credentials:'same-origin',headers:{'X-CSRF-Token':csrf,'Accept':'application/json'}});if(!response.ok)throw new Error('request');const result=await response.json();const resetURL=new URL(result.url);if(resetURL.protocol!=='https:'&&resetURL.protocol!=='http:'||resetURL.pathname!=='/password-reset'||resetURL.searchParams.get('token')?.length!==64)throw new Error('response');let copied=false;if(navigator.clipboard&&window.isSecureContext){try{await navigator.clipboard.writeText(result.url);copied=true}catch(_){}}if(status)status.textContent=copied?'Reset link copied. It expires in one hour.':result.url}catch(_){if(status)status.textContent='Could not create the reset link. Try again.'}finally{button.disabled=false}});`

const AdminDraftAutosaveJS = `(()=>{const scope=document.body?.dataset.studioUserScope;if(!scope)return;const saved=new URLSearchParams(location.search).get('saved')==='1';const forms=[...document.querySelectorAll('form[data-local-autosave][data-autosave-resource]')];if(!forms.length)return;const excluded=name=>name==='_csrf'||name==='version'||name.startsWith('block_csrf_');for(const form of forms){const key='hydro-draft:'+scope+':'+form.dataset.autosaveResource;let timer;const status=document.createElement('p');status.className='admin-draft-status';status.setAttribute('role','status');status.setAttribute('aria-live','polite');form.append(status);const controls=()=>[...form.elements].filter(control=>control.name&&!excluded(control.name)&&!['password','file','submit','button','reset'].includes(control.type));const snapshot=()=>{const fields={};for(const control of controls()){(fields[control.name]||(fields[control.name]=[])).push({type:control.type,value:control.value,checked:control.type==='checkbox'||control.type==='radio'?control.checked:undefined})}return{savedAt:Date.now(),fields}};const save=()=>{try{const value=JSON.stringify(snapshot());if(value.length>3*1024*1024){status.textContent='Yerel taslak sınırına ulaşıldı. Son değişiklikleri korumak için sayfayı kaydedin.';return}localStorage.setItem(key,value);status.textContent='Değişiklikler bu tarayıcıya kaydedildi.'}catch(_){status.textContent='Yerel kurtarma kullanılamıyor. Değişikliklerinizi kaydedin.'}};const schedule=()=>{clearTimeout(timer);timer=setTimeout(save,700)};form.addEventListener('input',schedule);form.addEventListener('change',schedule);if(saved){localStorage.removeItem(key);continue}let draft;try{draft=JSON.parse(localStorage.getItem(key)||'null')}catch(_){localStorage.removeItem(key)}if(!draft||!draft.savedAt||Date.now()-draft.savedAt>30*24*60*60*1000){if(draft)localStorage.removeItem(key);continue}const dialog=document.createElement('dialog');dialog.className='admin-draft-recovery';dialog.setAttribute('aria-labelledby','admin-draft-recovery-title');const heading=document.createElement('h2');heading.id='admin-draft-recovery-title';heading.textContent='Kaydedilmemiş taslak bulundu';const message=document.createElement('p');message.textContent=new Date(draft.savedAt).toLocaleString()+' tarihli yerel bir taslak var. Geri yükleyebilir veya silebilirsiniz.';const actions=document.createElement('div');actions.className='admin-draft-recovery__actions';const restore=document.createElement('button');restore.type='button';restore.className='button button-success';restore.textContent='Taslağı geri yükle';const discard=document.createElement('button');discard.type='button';discard.className='button button-quiet';discard.textContent='Sil';actions.append(restore,discard);dialog.append(heading,message,actions);document.body.append(dialog);restore.addEventListener('click',()=>{for(const [name,values] of Object.entries(draft.fields||{})){const matching=controls().filter(control=>control.name===name);values.forEach((savedValue,index)=>{const control=matching[index];if(!control)return;if(control.type==='checkbox'||control.type==='radio')control.checked=!!savedValue.checked;else control.value=savedValue.value;control.dispatchEvent(new Event('input',{bubbles:true}));control.dispatchEvent(new Event('change',{bubbles:true}))})}form.querySelectorAll('[data-richtext]').forEach(field=>{const value=field.querySelector('.richtext-value'),editor=field.querySelector('.richtext-editor');if(!value||!editor?.__quill)return;try{const parsed=JSON.parse(value.value);editor.__quill.setContents(parsed.document||parsed)}catch(_){editor.__quill.setText(value.value)}});status.textContent='Yerel taslak geri yüklendi. Değişiklikleri uygulamak için sayfayı kaydedin.';dialog.close();dialog.remove()});discard.addEventListener('click',()=>{localStorage.removeItem(key);dialog.close();dialog.remove()});dialog.addEventListener('cancel',()=>{localStorage.removeItem(key)});dialog.showModal()}})();`

const AdminPasswordStrengthJS = `(()=>{const common=['password','qwerty','welcome','admin','secret','letmein','iloveyou','monkey','dragon','football','abc123'];const score=value=>{const chars=Array.from(value),lower=value.toLowerCase();if(chars.length<8||chars.length>1024||new Set(chars).size<4||common.some(word=>lower.includes(word)))return 0;let pool=0,classes=0;if(/[a-z]/.test(value)){pool+=26;classes++}if(/[A-Z]/.test(value)){pool+=26;classes++}if(/[0-9]/.test(value)){pool+=10;classes++}if(/[^A-Za-z0-9]/.test(value)){pool+=32;classes++}const entropy=chars.length*Math.log2(pool||1);return classes&&entropy>=40?Math.min(4,1+Math.floor((entropy-40)/20)):0};const update=input=>{const form=input.closest('form'),meter=form?.querySelector('[data-password-meter]'),feedback=form?.querySelector('[data-password-feedback]');if(!meter)return;const value=score(input.value);meter.value=value;meter.classList.toggle('is-weak',value===0);meter.classList.toggle('is-fair',value>0&&value<3);meter.classList.toggle('is-strong',value>=3);if(feedback)feedback.textContent=value>=3?'Parola gücü: Güçlü.':value?'Parola gücü: Uygun. Daha uzun parolalar daha güvenlidir.':'En az 8 karakter seçin; yaygın parolalardan kaçının ve farklı karakterler ekleyin.'};document.addEventListener('input',event=>{const input=event.target.closest('[data-password-strength]');if(input)update(input)});document.addEventListener('submit',event=>{const form=event.target.closest('form'),input=form?.querySelector('[data-password-strength]');if(!input||score(input.value)>0)return;event.preventDefault();event.stopImmediatePropagation();const feedback=form.querySelector('[data-password-feedback]');if(feedback)feedback.textContent='Daha uzun ve tahmin edilmesi zor bir parola seçin.';input.focus()},true)})();`

const AdminPasswordSubmitJS = `(()=>{document.addEventListener('submit',async event=>{const form=event.target.closest('[data-quick-password-form]');if(!form)return;event.preventDefault();const status=form.querySelector('[data-quick-password-status]');if(form.elements.new_password.value!==form.elements.confirm_password.value){status.textContent='Yeni parolalar eşleşmiyor.';return}status.textContent='Parolanız güncelleniyor…';try{const csrf=decodeURIComponent((document.cookie.match(/(?:^|; )hydro_csrf=([^;]+)/)||[])[1]||'');const response=await fetch('/auth/me/credentials',{method:'POST',body:new FormData(form),headers:{'X-CSRF-Token':csrf}});if(!response.ok)throw new Error();if(form.elements.invalidate_other_sessions?.checked){status.textContent='Parolanız güncellendi; diğer cihazlardaki oturumlar kapatıldı.';form.reset();setTimeout(()=>{const dialog=form.closest('.admin-ux-dialog');if(dialog)dialog.hidden=true},1300)}else{status.textContent='Parolanız güncellendi. Yeniden giriş yapmanız gerekiyor.';setTimeout(()=>location.assign('/login'),700)}}catch{status.textContent='Parola güncellenemedi. Mevcut parolanızı kontrol edip yeni ve güçlü bir parola seçin.'}})})();`

const AdminListViewJS = `(()=>{const apply=(toggle,value)=>{const target=document.getElementById(toggle.dataset.listViewToggle||'');if(!target)return;const mode=value==='cards'?'cards':'table';target.dataset.viewMode=mode;toggle.querySelectorAll('[data-list-view-choice]').forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.listViewChoice===mode)))};const init=()=>document.querySelectorAll('[data-list-view-toggle]').forEach(toggle=>{const target=document.getElementById(toggle.dataset.listViewToggle||'');const key=toggle.dataset.storageKey||'hydro-view-default';let saved=target?.dataset.viewMode||'table';try{saved=localStorage.getItem(key)||saved}catch{}apply(toggle,saved)});if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',init,{once:true});else init();document.addEventListener('hydro:admin-navigated',init);document.addEventListener('click',event=>{const button=event.target.closest('[data-list-view-choice]');if(!button)return;const toggle=button.closest('[data-list-view-toggle]');if(!toggle)return;const value=button.dataset.listViewChoice==='cards'?'cards':'table';try{localStorage.setItem(toggle.dataset.storageKey||'hydro-view-default',value)}catch{}apply(toggle,value)})})();`

func adminUXScript() string {
	script := strings.Replace(AdminUXJS,
		`bar.innerHTML='<strong data-bulk-count></strong><button type="button" data-bulk-run>Uygula</button><button type="button" data-bulk-clear>Seçimi kaldır</button>'`,
		`bar.innerHTML='<strong data-bulk-count></strong><select data-bulk-choice aria-label="Toplu işlem seç"></select><button type="button" data-bulk-run>Uygula</button><button type="button" data-bulk-clear>Seçimi kaldır</button>'`, 1)
	script = strings.Replace(script,
		`qs('[data-bulk-count]',bar).textContent=selected.length+' kayıt seçildi'`,
		`qs('[data-bulk-count]',bar).textContent=selected.length+' kayıt seçildi';const choice=qs('[data-bulk-choice]',bar);if(choice&&!choice.options.length)qsa('[data-bulk-action]').forEach(action=>choice.add(new Option(action.textContent,action.dataset.bulkAction)))`, 1)
	script = strings.Replace(script,
		`const action=qs('[data-bulk-action]'),selected=boxes().filter(x=>x.checked)`,
		`const actions=qsa('[data-bulk-action]'),choice=qs('[data-bulk-choice]'),action=actions[choice?.selectedIndex||0],selected=boxes().filter(x=>x.checked)`, 1)
	script = strings.Replace(script,
		`if(e.target.closest('[data-bulk-action]')){e.preventDefault();qs('[data-bulk-run]')?.click()}`,
		`if(e.target.closest('[data-bulk-action]')){e.preventDefault();const action=e.target.closest('[data-bulk-action]'),actions=qsa('[data-bulk-action]'),choice=qs('[data-bulk-choice]');if(choice)choice.selectedIndex=actions.indexOf(action);qs('[data-bulk-run]')?.click()}`, 1)
	return script + ";\n" + AdminInviteJS
}

func RenderBusinessOwnerDashboard(w http.ResponseWriter, r *http.Request, enabled map[string]bool) error {
	return RenderBusinessOwnerDashboardWithSecurityNotice(w, r, enabled, "", "", 0)
}

func RenderBusinessOwnerDashboardWithSecurityNotice(w http.ResponseWriter, r *http.Request, enabled map[string]bool, previousLogin, previousIP string, failedAttempts int) error {
	user := auth.UserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleEditor || !user.Active {
		return errors.New("business dashboard is unavailable")
	}
	if err := AdminStartForRequest(w, r, "Hoş geldiniz", "", "dashboard"); err != nil {
		return err
	}
	if err := AdminLoginSecurityNotice(previousLogin, previousIP, failedAttempts).Render(r.Context(), w); err != nil {
		return err
	}
	if err := BusinessOwnerDashboard(enabled).Render(r.Context(), w); err != nil {
		return err
	}
	return AdminEnd(w)
}

const AdminInviteJS = `(()=>{const csrf=()=>decodeURIComponent((document.cookie.match(/(?:^|; )hydro_csrf=([^;]+)/)||[])[1]||'');document.addEventListener('submit',async event=>{const form=event.target.closest('form[data-admin-invite]');if(!form)return;event.preventDefault();const status=form.querySelector('[data-invite-status]'),result=form.querySelector('[data-invite-result]'),input=form.querySelector('[data-invite-link]');if(!status||!result||!input)return;status.textContent='Davet bağlantısı oluşturuluyor…';result.hidden=true;const submit=form.querySelector('[type=submit]');if(submit)submit.disabled=true;try{const response=await fetch(form.action,{method:'POST',body:new FormData(form),credentials:'same-origin',headers:{'X-CSRF-Token':csrf(),'Accept':'application/json'}});if(!response.ok)throw new Error('invite');const payload=await response.json();const url=new URL(payload.url,location.origin);if(url.origin!==location.origin||url.pathname!='/auth/invites/accept'||!url.searchParams.has('token'))throw new Error('invite');input.value=url.href;result.hidden=false;status.textContent='Davet bağlantısı oluşturuldu. Bağlantıyı ekip üyesiyle paylaşın.'}catch{status.textContent='Davet oluşturulamadı. E-posta adresini ve gönderim ayarlarını kontrol edin.'}finally{if(submit)submit.disabled=false}});document.addEventListener('click',async event=>{const button=event.target.closest('[data-invite-copy]');if(!button)return;const form=button.closest('form[data-admin-invite]'),input=form?.querySelector('[data-invite-link]'),status=form?.querySelector('[data-invite-status]');if(!input?.value||!status)return;try{await navigator.clipboard.writeText(input.value);status.textContent='Davet bağlantısı panoya kopyalandı.'}catch{input.focus();input.select();status.textContent='Bağlantı seçildi. Kopyalamak için Ctrl+C tuşlarına basın.'}})})();`
