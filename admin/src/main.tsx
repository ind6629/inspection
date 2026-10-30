import React, {useEffect,useState} from 'react';
import {createRoot} from 'react-dom/client';
import './style.css';
type Status='open'|'in_progress'|'resolved';
type Issue={id:string;title:string;description:string;priority:'low'|'medium'|'high';status:Status;position:{x:number;y:number;z:number};createdAt:string;updatedAt:string};
const labels:Record<Status,string>={open:'待处理',in_progress:'处理中',resolved:'已解决'};
const priorities={low:'低',medium:'中',high:'高'};
const base=(import.meta as ImportMeta & {env:Record<string,string>}).env.VITE_API_BASE_URL||'http://localhost:8080';
async function api<T>(path:string,init?:RequestInit):Promise<T>{
 const response=await fetch(base+path,{...init,signal:AbortSignal.timeout(10000)});
 const data=await response.json();if(!response.ok)throw Error(data.error?.message||`HTTP ${response.status}`);return data;
}
function App(){
 const [issues,setIssues]=useState<Issue[]>([]),[loading,setLoading]=useState(false),[error,setError]=useState(''),[saving,setSaving]=useState(''),[filter,setFilter]=useState('all'),[search,setSearch]=useState(''),[selected,setSelected]=useState('');
 async function load(){setLoading(true);setError('');try{setIssues(await api<Issue[]>('/api/issues'));}catch(e){setError('无法加载问题，请确认 Go 服务已启动。'+String(e));}finally{setLoading(false);}}
 useEffect(()=>{void load();},[]);
 async function update(id:string,status:Status){setSaving(id);setError('');try{const result=await api<Issue>('/api/issues/'+encodeURIComponent(id)+'/status',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({status})});setIssues(old=>old.map(i=>i.id===id?result:i));}catch(e){setError('修改失败，原状态已保留。'+String(e));}finally{setSaving('');}}
 const shown=issues.filter(i=>(filter==='all'||i.status===filter)&&i.title.toLowerCase().includes(search.toLowerCase())).slice().reverse();
 const detail=issues.find(i=>i.id===selected);
 return <main><header><div><small>FIELD OPERATIONS / 虚拟现场</small><h1>巡检问题管理</h1><p>记录每一个现场问题，跟进每一次处理。</p></div><button disabled={loading||!!saving} onClick={()=>void load()}>{loading?'加载中…':'刷新问题'}</button></header>
 <section className="stats">{(['all','open','in_progress','resolved'] as const).map(s=><button className={filter===s?'active':''} onClick={()=>setFilter(s)} key={s}><span>{s==='all'?'全部问题':labels[s]}</span><strong>{issues.filter(i=>s==='all'||i.status===s).length}</strong></button>)}</section>
 {error&&<div role="alert" className="error">{error} <button disabled={loading} onClick={()=>void load()}>重试</button></div>}
 <section className="panel"><div className="toolbar"><h2>问题清单 <small>{shown.length} 条</small></h2><input aria-label="搜索标题" placeholder="搜索问题标题…" value={search} onChange={e=>setSearch(e.target.value)}/></div>
 <div className="table"><table><thead><tr><th>问题</th><th>优先级</th><th>处理状态</th><th>上报时间</th><th>详情</th></tr></thead><tbody>{shown.map(i=><tr key={i.id}><td><b>{i.title}</b><small className="id">{i.id.slice(0,8)}</small></td><td><span className={'badge '+i.priority}>{priorities[i.priority]}优先级</span></td><td><select aria-label={i.title+'的状态'} disabled={!!saving} value={i.status} onChange={e=>void update(i.id,e.target.value as Status)}>{Object.entries(labels).map(([value,label])=><option key={value} value={value}>{label}</option>)}</select>{saving===i.id&&' 保存中…'}</td><td>{new Date(i.createdAt).toLocaleString()}</td><td><button onClick={()=>setSelected(i.id)}>查看</button></td></tr>)}</tbody></table></div>
 {!loading&&!shown.length&&<div className="empty">{issues.length?'没有符合条件的问题':'暂无问题，请在 Unity 中点击设备并提交巡检问题。'}</div>}</section>
 {detail&&<div className="overlay" onClick={()=>setSelected('')}><section role="dialog" aria-modal="true" aria-label="问题详情" className="dialog" onClick={e=>e.stopPropagation()}><button className="close" onClick={()=>setSelected('')}>关闭</button><small>问题详情</small><h2>{detail.title}</h2><p className="description">{detail.description||'暂无描述'}</p><dl><dt>优先级 / 状态</dt><dd>{priorities[detail.priority]} / {labels[detail.status]}</dd><dt>空间位置（米）</dt><dd>{Object.entries(detail.position).map(([k,v])=>`${k}: ${v.toFixed(3)}`).join(' / ')}</dd><dt>最后更新</dt><dd>{new Date(detail.updatedAt).toLocaleString()}</dd><dt>问题 ID</dt><dd>{detail.id}</dd></dl></section></div>}
 <footer>本地巡检演示 · 数据由同一个 Go 服务持久化保存</footer></main>;
}
createRoot(document.getElementById('root')!).render(<App/>);
