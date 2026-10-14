(()=>{
const schemas={hero:['type','title','subtitle','media','buttonText','buttonURL'],cta:['type','title','subtitle','media','buttonText','buttonURL'],faq:['type','items'],pricing_table:['type','items'],testimonials:['type','items'],before_after:['type','payload'],stats:['type','payload'],contact_cta:['type','payload'],video:['type','payload'],masonry:['type','payload'],hotspots:['type','payload'],callout:['type','payload'],action_card:['type','payload'],carousel:['type','payload'],columns:['type','payload'],feature_list:['type','payload'],video_hero:['type','payload'],trust_bar:['type','payload'],figure:['type','payload'],business_status:['type','payload']};
const rows=()=>Array.from(document.querySelectorAll('[data-block-row]'));

const initFAQBuilder=(field,textarea)=>{
	if(field.querySelector('.faq-visual-builder'))return;
	const container=document.createElement('div');
	container.className='faq-visual-builder';
	let items=[];
	try{items=JSON.parse(textarea.value||'[]');if(!Array.isArray(items))items=[];}catch(_){items=[];}
	const listWrap=document.createElement('div');
	listWrap.className='faq-items-list';
	const syncToTextarea=()=>{
		const cards=listWrap.querySelectorAll('.faq-item-card');
		const result=[];
		cards.forEach(card=>{
			const q=card.querySelector('.faq-q-input')?.value.trim()||'';
			const a=card.querySelector('.faq-a-input')?.value.trim()||'';
			if(q||a)result.push({question:q,answer:a});
		});
		textarea.value=JSON.stringify(result);
		textarea.dispatchEvent(new Event('input',{bubbles:true}));
	};
	const updateCardNumbers=()=>{
		listWrap.querySelectorAll('.faq-item-card').forEach((card,idx)=>{
			const t=card.querySelector('.faq-card-title');
			if(t)t.textContent='Soru #'+(idx+1);
		});
	};
	const createCard=(question='',answer='')=>{
		const card=document.createElement('div');
		card.className='faq-item-card';
		const header=document.createElement('div');
		header.className='faq-item-card-header';
		const titleSpan=document.createElement('strong');
		titleSpan.className='faq-card-title';
		titleSpan.textContent='Soru';
		const delBtn=document.createElement('button');
		delBtn.type='button';
		delBtn.className='faq-item-remove-btn';
		delBtn.innerHTML='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg> Sil';
		delBtn.addEventListener('click',()=>{card.remove();syncToTextarea();updateCardNumbers();});
		header.append(titleSpan,delBtn);
		const body=document.createElement('div');
		body.className='faq-item-card-body';
		const qLabel=document.createElement('label');
		qLabel.className='faq-field-label';
		qLabel.innerHTML='<span>Soru</span>';
		const qInput=document.createElement('input');
		qInput.type='text';
		qInput.className='faq-q-input';
		qInput.placeholder='Örn: Destek ve kurulum süreci nasıl işliyor?';
		qInput.value=question;
		qInput.addEventListener('input',syncToTextarea);
		qLabel.appendChild(qInput);
		const aLabel=document.createElement('label');
		aLabel.className='faq-field-label';
		aLabel.innerHTML='<span>Cevap</span>';
		const aInput=document.createElement('textarea');
		aInput.className='faq-a-input';
		aInput.rows=2;
		aInput.placeholder='Örn: Kurulum 15 dakika içinde tamamlanır ve teknik ekibimiz 7/24 destek sağlar.';
		aInput.value=answer;
		aInput.addEventListener('input',syncToTextarea);
		aLabel.appendChild(aInput);
		body.append(qLabel,aLabel);
		card.append(header,body);
		listWrap.appendChild(card);
		syncToTextarea();
	};
	if(items.length>0){items.forEach(it=>createCard(it.question||'',it.answer||''));}else{createCard('','');}
	updateCardNumbers();
	const footer=document.createElement('div');
	footer.className='faq-builder-footer';
	const addBtn=document.createElement('button');
	addBtn.type='button';
	addBtn.className='button button-quiet button-sm faq-add-card-btn';
	addBtn.innerHTML='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14"/><path d="M5 12h14"/></svg> Yeni Soru Ekle';
	addBtn.addEventListener('click',()=>{
		createCard('','');
		updateCardNumbers();
		const inputs=listWrap.querySelectorAll('.faq-q-input');
		if(inputs.length)inputs[inputs.length-1].focus();
	});
	const jsonToggle=document.createElement('button');
	jsonToggle.type='button';
	jsonToggle.className='faq-json-toggle-btn';
	jsonToggle.textContent='JSON Görünümü';
	jsonToggle.addEventListener('click',()=>{
		const isHidden=textarea.style.display==='none'||textarea.hidden;
		if(isHidden){
			textarea.style.display='block';
			textarea.hidden=false;
			jsonToggle.textContent='Görsel Formu Göster';
		}else{
			textarea.style.display='none';
			textarea.hidden=true;
			jsonToggle.textContent='JSON Görünümü';
			try{
				const parsed=JSON.parse(textarea.value||'[]');
				if(Array.isArray(parsed)){
					listWrap.innerHTML='';
					parsed.forEach(p=>createCard(p.question||'',p.answer||''));
					updateCardNumbers();
				}
			}catch(_){}
		}
	});
	footer.append(addBtn,jsonToggle);
	container.append(listWrap,footer);
	textarea.style.display='none';
	textarea.hidden=true;
	field.appendChild(container);
};

const initTestimonialsBuilder=(field,textarea)=>{
	if(field.querySelector('.testimonials-visual-builder'))return;
	const container=document.createElement('div');
	container.className='testimonials-visual-builder';
	let items=[];
	try{items=JSON.parse(textarea.value||'[]');if(!Array.isArray(items))items=[];}catch(_){items=[];}
	const listWrap=document.createElement('div');
	listWrap.className='testimonial-items-list';
	const syncToTextarea=()=>{
		const cards=listWrap.querySelectorAll('.testimonial-item-card');
		const result=[];
		cards.forEach(card=>{
			const n=card.querySelector('.testimonial-n-input')?.value.trim()||'';
			const q=card.querySelector('.testimonial-q-input')?.value.trim()||'';
			if(n||q)result.push({name:n,quote:q});
		});
		textarea.value=JSON.stringify(result);
		textarea.dispatchEvent(new Event('input',{bubbles:true}));
	};
	const updateCardNumbers=()=>{
		listWrap.querySelectorAll('.testimonial-item-card').forEach((card,idx)=>{
			const t=card.querySelector('.testimonial-card-title');
			if(t)t.textContent='Yorum #'+(idx+1);
		});
	};
	const createCard=(name='',quote='')=>{
		const card=document.createElement('div');
		card.className='testimonial-item-card';
		const header=document.createElement('div');
		header.className='faq-item-card-header';
		const titleSpan=document.createElement('strong');
		titleSpan.className='testimonial-card-title';
		titleSpan.textContent='Yorum';
		const delBtn=document.createElement('button');
		delBtn.type='button';
		delBtn.className='faq-item-remove-btn';
		delBtn.innerHTML='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg> Sil';
		delBtn.addEventListener('click',()=>{card.remove();syncToTextarea();updateCardNumbers();});
		header.append(titleSpan,delBtn);
		const body=document.createElement('div');
		body.className='faq-item-card-body';
		const nLabel=document.createElement('label');
		nLabel.className='faq-field-label';
		nLabel.innerHTML='<span>Müşteri Adı / Unvanı</span>';
		const nInput=document.createElement('input');
		nInput.type='text';
		nInput.className='testimonial-n-input';
		nInput.placeholder='Örn: Burak Kaya - Kurucu Ortak';
		nInput.value=name;
		nInput.addEventListener('input',syncToTextarea);
		nLabel.appendChild(nInput);
		const qLabel=document.createElement('label');
		qLabel.className='faq-field-label';
		qLabel.innerHTML='<span>Görüş / Yorum</span>';
		const qInput=document.createElement('textarea');
		qInput.className='testimonial-q-input';
		qInput.rows=2;
		qInput.placeholder='Örn: HydroGo platformu ile iş süreçlerimizi hızlandırdık.';
		qInput.value=quote;
		qInput.addEventListener('input',syncToTextarea);
		qLabel.appendChild(qInput);
		body.append(nLabel,qLabel);
		card.append(header,body);
		listWrap.appendChild(card);
		syncToTextarea();
	};
	if(items.length>0){items.forEach(it=>createCard(it.name||'',it.quote||''));}else{createCard('','');}
	updateCardNumbers();
	const footer=document.createElement('div');
	footer.className='faq-builder-footer';
	const addBtn=document.createElement('button');
	addBtn.type='button';
	addBtn.className='button button-quiet button-sm faq-add-card-btn';
	addBtn.innerHTML='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14"/><path d="M5 12h14"/></svg> Yeni Yorum Ekle';
	addBtn.addEventListener('click',()=>{
		createCard('','');
		updateCardNumbers();
		const inputs=listWrap.querySelectorAll('.testimonial-n-input');
		if(inputs.length)inputs[inputs.length-1].focus();
	});
	const jsonToggle=document.createElement('button');
	jsonToggle.type='button';
	jsonToggle.className='faq-json-toggle-btn';
	jsonToggle.textContent='JSON Görünümü';
	jsonToggle.addEventListener('click',()=>{
		const isHidden=textarea.style.display==='none'||textarea.hidden;
		if(isHidden){
			textarea.style.display='block';
			textarea.hidden=false;
			jsonToggle.textContent='Görsel Formu Göster';
		}else{
			textarea.style.display='none';
			textarea.hidden=true;
			jsonToggle.textContent='JSON Görünümü';
			try{
				const parsed=JSON.parse(textarea.value||'[]');
				if(Array.isArray(parsed)){
					listWrap.innerHTML='';
					parsed.forEach(p=>createCard(p.name||'',p.quote||''));
					updateCardNumbers();
				}
			}catch(_){}
		}
	});
	footer.append(addBtn,jsonToggle);
	container.append(listWrap,footer);
	textarea.style.display='none';
	textarea.hidden=true;
	field.appendChild(container);
};

const sync=row=>{
	const select=row.querySelector('select[name^="block_type_"]');
	if(!select)return;
	const val=select.value;
	const fields=new Set(schemas[val]||['type']);
	row.dataset.blockType=val;
	const badge=row.querySelector('[data-block-type-badge]');
	if(badge){
		const opt=select.options[select.selectedIndex];
		badge.textContent=val?(opt?.text.split('(')[0].trim()||val):'Boş Blok';
		badge.style.display=val?'inline-block':'none';
	}
	row.querySelectorAll('[data-block-field]').forEach(field=>{
		field.hidden=!fields.has(field.dataset.blockField);
	});
	if(val==='faq'){
		const itemsField=row.querySelector('[data-block-field="items"]');
		const textarea=itemsField?.querySelector('textarea[name^="block_items_"]');
		if(itemsField&&textarea)initFAQBuilder(itemsField,textarea);
	}else if(val==='testimonials'){
		const itemsField=row.querySelector('[data-block-field="items"]');
		const textarea=itemsField?.querySelector('textarea[name^="block_items_"]');
		if(itemsField&&textarea)initTestimonialsBuilder(itemsField,textarea);
	}
};

const reindex=()=>{
	const list=rows();
	list.forEach((row,index)=>{
		row.dataset.blockIndex=String(index);
		row.querySelectorAll('[name]').forEach(input=>{
			input.name=input.name.replace(/^(block_[a-z_]+)_\d+$/,'$1_'+index);
		});
		row.querySelectorAll('[id]').forEach(input=>{
			input.id=input.id.replace(/^(block_[a-z_]+)_\d+$/,'$1_'+index);
		});
		const target=row.querySelector('[data-media-target]');
		if(target)target.dataset.mediaTarget='block_media_asset_id_'+index;
		const heading=row.querySelector('[data-block-number]');
		if(heading)heading.textContent='Blok '+(index+1);
		const upBtn=row.querySelector('[data-block-up]');
		if(upBtn)upBtn.disabled=index===0;
		const downBtn=row.querySelector('[data-block-down]');
		if(downBtn)downBtn.disabled=index===list.length-1;
	});
	const count=document.querySelector('[name="block_count"]');
	if(count)count.value=String(list.length);
};

window.HydroInitBlockEditor=()=>{
	const root=document.querySelector('[data-block-editor]');
	if(!root||root.dataset.ready)return;
	root.dataset.ready='true';
	root.addEventListener('change',event=>{
		if(event.target.matches('select[name^="block_type_"]'))sync(event.target.closest('[data-block-row]'));
	});
	root.addEventListener('click',event=>{
		const button=event.target.closest('[data-block-add],[data-block-up],[data-block-down],[data-block-remove]');
		if(!button)return;
		event.preventDefault();
		const list=rows();
		if(button.matches('[data-block-add]')){
			if(list.length>=64)return;
			const last=list.at(-1);
			const row=last.cloneNode(true);
			row.querySelectorAll('input:not([type="hidden"]),textarea').forEach(input=>input.value='');
			row.querySelectorAll('select').forEach(select=>select.value='');
			row.querySelectorAll('.faq-visual-builder,.testimonials-visual-builder').forEach(el=>el.remove());
			row.dataset.blockInit='';
			root.insertBefore(row,root.querySelector('[data-block-add]'));
			sync(row);
			reindex();
		}else if(button.matches('[data-block-remove]')){
			const row=button.closest('[data-block-row]');
			if(list.length>1){
				row.remove();
				reindex();
			}else{
				const sel=row.querySelector('select[name^="block_type_"]');
				if(sel){sel.value='';sync(row);}
				row.querySelectorAll('input:not([type="hidden"]),textarea').forEach(input=>input.value='');
			}
		}else{
			const row=button.closest('[data-block-row]');
			const index=list.indexOf(row);
			const offset=button.matches('[data-block-up]')?-1:1;
			const other=list[index+offset];
			if(!other)return;
			offset<0?root.insertBefore(row,other):root.insertBefore(other,row);
			reindex();
		}
	});
	root.querySelectorAll('[data-block-row]').forEach(sync);
	reindex();
};
let editorJSLoading;
window.HydroInitEditorJS=config=>{
	const mount=document.getElementById(config.mount),input=document.getElementById(config.input);
	if(!mount||!input||mount.dataset.ready)return;
	const init=()=>{
		if(!window.EditorJS||mount.dataset.ready)return;
		mount.dataset.ready='true';
		let data={};try{data=JSON.parse(input.value||'{}')}catch(_e){}
		let editor;
		if(Array.isArray(data.blocks)&&data.blocks.length===0)data={blocks:[{type:'paragraph',data:{text:''}}]};
		const labelBlocks=()=>mount.querySelectorAll('[contenteditable="true"]').forEach((block,index)=>{block.setAttribute('role','textbox');block.setAttribute('aria-multiline','true');block.setAttribute('aria-label',(mount.dataset.editorjsLabel||'İçerik')+' · '+(index+1));});
		editor=new window.EditorJS({holder:mount.id,data:data,onReady:()=>{labelBlocks();new MutationObserver(labelBlocks).observe(mount,{childList:true,subtree:true});},onChange:async()=>{try{input.value=JSON.stringify(await editor.save());input.dispatchEvent(new Event('keyup',{bubbles:true}));}catch(_e){}}});
	};
	if(window.EditorJS){init();return;}
	if(!editorJSLoading){
		editorJSLoading=new Promise((resolve,reject)=>{
			const script=document.createElement('script');script.src=mount.dataset.editorjsScript;script.onload=resolve;script.onerror=reject;document.head.append(script);
		});
	}
	editorJSLoading.then(init).catch(()=>{mount.textContent='Blok düzenleyicisi yüklenemedi.';mount.setAttribute('role','alert');});
};
window.HydroInitGeneratedBlockEditors=()=>{
	const mounts=Array.from(document.querySelectorAll('[data-editorjs-mount]'));
	const needsPreview=mounts.some(mount=>mount.dataset.editorjsPreview==='true');
	const initialize=()=>mounts.forEach(mount=>window.HydroInitEditorJS({mount:mount.id,input:mount.dataset.editorjsMount}));
	if(!needsPreview||window.htmx){initialize();return;}
	let script=document.querySelector('script[data-hydro-block-editor-htmx]');
	if(script){script.addEventListener('load',initialize,{once:true});return;}
	script=document.createElement('script');
	script.src='/vendor/htmx/htmx.min.js';
	script.dataset.hydroBlockEditorHtmx='true';
	script.addEventListener('load',()=>{window.htmx?.process(document.body);initialize();},{once:true});
	script.addEventListener('error',()=>mounts.forEach(mount=>{mount.dataset.editorjsError='preview-runtime-unavailable';}),{once:true});
	document.head.appendChild(script);
};
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',window.HydroInitBlockEditor);else window.HydroInitBlockEditor();
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',window.HydroInitGeneratedBlockEditors);else window.HydroInitGeneratedBlockEditors();
})()
