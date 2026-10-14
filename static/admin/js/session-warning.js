(()=>{
if(!location.pathname.startsWith('/admin'))return;
const english=document.documentElement.lang==='en';
let timer;
const start=async()=>{
	try{
		const response=await fetch('/auth/me',{credentials:'same-origin',headers:{'Accept':'application/json'}});
		if(!response.ok)return;
		const data=await response.json();
		const expires=Date.parse(data.session_expires_at||'');
		if(!Number.isFinite(expires))return;
		clearTimeout(timer);
		timer=setTimeout(show,Math.max(0,expires-Date.now()-120000));
	}catch(_){}
};
const show=()=>{
	if(!document.body||document.getElementById('hydro-session-warning'))return;
	const dialog=document.createElement('section');
	dialog.id='hydro-session-warning';
	dialog.setAttribute('role','alertdialog');
	dialog.setAttribute('aria-modal','true');
	dialog.setAttribute('aria-labelledby','hydro-session-warning-title');
	dialog.style.cssText='position:fixed;z-index:10000;inset:auto 1rem 1rem auto;max-width:28rem;padding:1rem;background:var(--card,#fff);color:var(--foreground,#111);border:1px solid var(--border,#ddd);border-radius:.75rem;box-shadow:0 8px 32px #0003';
	const title=document.createElement('h2');
	title.id='hydro-session-warning-title';
	title.textContent=english?'Your session will expire soon':'Oturumunuz yakında sona erecek';
	const copy=document.createElement('p');
	copy.textContent=english?'Extend your session to keep working.':'Çalışmanızı sürdürmek için oturumunuzu uzatabilirsiniz.';
	const extend=document.createElement('button');
	extend.type='button';
	extend.textContent=english?'Extend session':'Oturumu uzat';
	const countdown=document.createElement('span');
	countdown.setAttribute('aria-live','polite');
	countdown.style.marginLeft='.75rem';
	dialog.append(title,copy,extend,countdown);
	document.body.append(dialog);
	let remaining=120;
	const tick=()=>{
		countdown.textContent=Math.floor(remaining/60)+':'+String(remaining%60).padStart(2,'0');
		if(remaining--<=0)window.location.assign('/login?return_to='+encodeURIComponent(location.pathname+location.search));
	};
	tick();
	const interval=setInterval(tick,1000);
	extend.addEventListener('click',async()=>{
		extend.disabled=true;
		const csrf=document.cookie.split('; ').find(row=>row.startsWith('hydro_csrf='));
		try{
			const result=await fetch('/auth/me/session/extend',{method:'POST',credentials:'same-origin',headers:{'X-CSRF-Token':csrf?decodeURIComponent(csrf.split('=').slice(1).join('=')):''}});
			if(!result.ok)throw new Error('session');
			clearInterval(interval);
			dialog.remove();
			start();
		}catch(_){
			extend.disabled=false;
			extend.textContent=english?'Try again':'Tekrar dene';
		}
	});
	extend.focus();
};
const install=()=>{if(document.body)start();};
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',install,{once:true});else install();
})();
