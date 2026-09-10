import { StrictMode, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import "./style.css";

type NewsItem = { id:string; slug:string; title:string; body:string; status:string; revision:number; updated_at:string };
function App() {
  const [items,setItems]=useState<NewsItem[]>([]); const [query,setQuery]=useState(""); const [selected,setSelected]=useState<NewsItem|null>(null); const [state,setState]=useState("loading");
  useEffect(()=>{fetch("/api/v1/news?locale=tr&limit=50").then(r=>r.ok?r.json():Promise.reject()).then(v=>{setItems(v);setState("ready")}).catch(()=>setState("error"))},[]);
  const filtered=useMemo(()=>items.filter(x=>(x.title+" "+x.body).toLowerCase().includes(query.toLowerCase())),[items,query]);
  return <div className="app"><aside><div className="brand">HYDRO<span>GO</span><small>REACT STUDY</small></div><nav><a className="active">◈ Haberler</a><a>▤ Sayfalar</a><a>◫ Commerce</a></nav></aside><main><header><div className="search">⌕ <input value={query} onChange={e=>setQuery(e.target.value)} placeholder="Haberlerde ara..." /></div><span className="user">● Admin</span></header><section className="heading"><div><p>YAYIN MERKEZİ</p><h1>Haberler</h1><span>JSON API + client state ile zengin içerik yönetimi.</span></div><button>+ Yeni haber</button></section><section className="card"><div className="card-head"><h2>İçerik akışı <em>{filtered.length}</em></h2><span>React · local filtering</span></div>{state==="loading"&&<div className="loading">Haberler yükleniyor...</div>}{state==="error"&&<div className="alert">Haber API’sine bağlanılamadı.</div>}{state==="ready"&&<div className="list">{filtered.map((x,i)=><button className="row" key={x.id} onClick={()=>setSelected(x)}><span className="num">{String(i+1).padStart(2,"0")}</span><span><strong>{x.title}</strong><small>{x.body.replace(/\s+/g," ").slice(0,120)}</small></span><span className="status">{x.status}</span></button>)}</div>}</section>{selected&&<div className="modal" onClick={()=>setSelected(null)}><article onClick={e=>e.stopPropagation()}><button className="close" onClick={()=>setSelected(null)}>×</button><p>HABER DETAYI</p><h2>{selected.title}</h2><div>{selected.body}</div><footer>Revizyon {selected.revision}</footer></article></div>}</main></div>
}
createRoot(document.getElementById("root")!).render(<StrictMode><App/></StrictMode>);
