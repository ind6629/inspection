import React, {useEffect,useState} from 'react';
import {createRoot} from 'react-dom/client';
import './style.css';
type Status='open'|'in_progress'|'resolved';
type Issue={id:string;title:string;description:string;priority:'low'|'medium'|'high';status:Status;position:{x:number;y:number;z:number};createdAt:string;updatedAt:string};
const labels:Record<Status,string>={open:'待处理',in_progress:'处理中',resolved:'已解决'};
const priorities={low:'低',medium:'中',high:'高'};
const base=(import.meta as ImportMeta & {env:Record<string,string>}).env.VITE_API_BASE_URL||`http://${window.location.hostname}:8080`;
class ApiError extends Error { constructor(public status:number,message:string){super(message);} }
async function api<T>(path:string,init?:RequestInit):Promise<T>{
 const response=await fetch(base+path,{...init,credentials:'include',signal:AbortSignal.timeout(10000)});
 const data=await response.json();if(!response.ok)throw new ApiError(response.status,data.error?.message||`HTTP ${response.status}`);return data;
}
function Dashboard({username,onLogout,onExpired}:{username:string;onLogout:()=>void;onExpired:()=>void}){
 const [issues,setIssues]=useState<Issue[]>([]),[loading,setLoading]=useState(false),[error,setError]=useState(''),[saving,setSaving]=useState(''),[filter,setFilter]=useState('all'),[search,setSearch]=useState(''),[selected,setSelected]=useState('');
 async function load(){setLoading(true);setError('');try{setIssues(await api<Issue[]>('/api/admin/issues'));}catch(e){if(e instanceof ApiError&&e.status===401){onExpired();return;}setError('无法加载问题，请确认 Go 服务已启动。'+String(e));}finally{setLoading(false);}}
 useEffect(()=>{void load();},[]);
 async function update(id:string,status:Status){setSaving(id);setError('');try{const result=await api<Issue>('/api/issues/'+encodeURIComponent(id)+'/status',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({status})});setIssues(old=>old.map(i=>i.id===id?result:i));}catch(e){if(e instanceof ApiError&&e.status===401){onExpired();return;}setError('修改失败，原状态已保留。'+String(e));}finally{setSaving('');}}
 const shown=issues.filter(i=>(filter==='all'||i.status===filter)&&i.title.toLowerCase().includes(search.toLowerCase())).slice().reverse();
 const detail=issues.find(i=>i.id===selected);
 return <main><header><div><small>现场巡检 / 管理工作台</small><h1>巡检问题管理</h1><p>记录每一个现场问题，跟进每一次处理。</p></div><div className="account"><span>管理员 · {username}</span><button disabled={loading||!!saving} onClick={()=>void load()}>{loading?'加载中…':'刷新问题'}</button><button disabled={!!saving} onClick={onLogout}>退出登录</button></div></header>
 <section className="stats">{(['all','open','in_progress','resolved'] as const).map(s=><button className={filter===s?'active':''} onClick={()=>setFilter(s)} key={s}><span>{s==='all'?'全部问题':labels[s]}</span><strong>{issues.filter(i=>s==='all'||i.status===s).length}</strong></button>)}</section>
 {error&&<div role="alert" className="error">{error} <button disabled={loading} onClick={()=>void load()}>重试</button></div>}
 <section className="panel"><div className="toolbar"><h2>问题清单 <small>{shown.length} 条</small></h2><input aria-label="搜索标题" placeholder="搜索问题标题…" value={search} onChange={e=>setSearch(e.target.value)}/></div>
 <div className="table"><table><thead><tr><th>问题</th><th>优先级</th><th>处理状态</th><th>上报时间</th><th>详情</th></tr></thead><tbody>{shown.map(i=><tr key={i.id}><td><b>{i.title}</b><small className="id">{i.id.slice(0,8)}</small></td><td><span className={'badge '+i.priority}>{priorities[i.priority]}优先级</span></td><td><select aria-label={i.title+'的状态'} disabled={!!saving} value={i.status} onChange={e=>void update(i.id,e.target.value as Status)}>{Object.entries(labels).map(([value,label])=><option key={value} value={value}>{label}</option>)}</select>{saving===i.id&&' 保存中…'}</td><td>{new Date(i.createdAt).toLocaleString()}</td><td><button onClick={()=>setSelected(i.id)}>查看</button></td></tr>)}</tbody></table></div>
 {!loading&&!shown.length&&<div className="empty">{issues.length?'没有符合条件的问题':'暂无问题，请在 Unity 中点击设备并提交巡检问题。'}</div>}</section>
 {detail&&<div className="overlay" onClick={()=>setSelected('')}><section role="dialog" aria-modal="true" aria-label="问题详情" className="dialog" onClick={e=>e.stopPropagation()}><button className="close" onClick={()=>setSelected('')}>关闭</button><small>问题详情</small><h2>{detail.title}</h2><p className="description">{detail.description||'暂无描述'}</p><dl><dt>优先级 / 状态</dt><dd>{priorities[detail.priority]} / {labels[detail.status]}</dd><dt>空间位置（米）</dt><dd>{Object.entries(detail.position).map(([k,v])=>`${k}: ${v.toFixed(3)}`).join(' / ')}</dd><dt>最后更新</dt><dd>{new Date(detail.updatedAt).toLocaleString()}</dd><dt>问题 ID</dt><dd>{detail.id}</dd></dl></section></div>}
 <footer>本地巡检演示 · 数据由同一个 Go 服务持久化保存</footer></main>;
}
function App(){
 const [user,setUser]=useState<string|null>(null),[checking,setChecking]=useState(true),[username,setUsername]=useState('admin'),[password,setPassword]=useState(''),[busy,setBusy]=useState(false),[error,setError]=useState(''),[visible,setVisible]=useState(false);
 async function check(){setChecking(true);setError('');try{const me=await api<{username:string}>('/api/auth/me');setUser(me.username);}catch(e){if(!(e instanceof ApiError&&e.status===401))setError('无法连接后端，请启动 Go 服务后重试。');}finally{setChecking(false);}}
 useEffect(()=>{void check();},[]);
 async function login(event:React.FormEvent){event.preventDefault();setBusy(true);setError('');try{const me=await api<{username:string}>('/api/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:username.trim(),password})});setPassword('');setUser(me.username);}catch(e){setError(e instanceof ApiError?e.message:'无法连接后端，请确认 Go 服务已启动。');}finally{setBusy(false);}}
 async function logout(){setBusy(true);try{await api('/api/auth/logout',{method:'POST'});setUser(null);setPassword('');setError('');}catch{setError('退出失败，请检查后端后重试。');}finally{setBusy(false);}}
 if(checking)return <div className="auth-loading" role="status">正在检查登录状态…</div>;
 if(user)return <>{error&&<div role="alert" className="error">{error}</div>}<Dashboard username={user} onLogout={()=>{if(!busy)void logout();}} onExpired={()=>{setUser(null);setError('登录已过期，请重新登录。');}}/></>;
 return <div className="login-page"><aside className="login-intro"><div className="brand-icon">巡</div><span className="eyebrow">虚拟现场巡检平台</span><h1>现场问题，<br/>有迹可循。</h1><p>从空间标注到处理闭环，<br/>让每一次巡检都有清晰的记录。</p><div className="login-flow"><span>01　现场标注</span><span>02　问题上报</span><span>03　跟进处理</span></div><small>Unity × Go × React · 本地演示系统</small></aside><section className="login-side"><form className="login-card" onSubmit={e=>void login(e)}><span className="eyebrow">管理工作台</span><h2>欢迎登录</h2><p>使用管理员账号查看问题并更新处理状态。</p><label htmlFor="username">管理员账号</label><input id="username" autoComplete="username" required maxLength={100} disabled={busy} value={username} onChange={e=>setUsername(e.target.value)}/><label htmlFor="password">登录密码</label><div className="password-field"><input id="password" type={visible?'text':'password'} autoComplete="current-password" required disabled={busy} value={password} onChange={e=>setPassword(e.target.value)}/><button type="button" aria-label={visible?'隐藏密码':'显示密码'} onClick={()=>setVisible(v=>!v)}>{visible?'隐藏':'显示'}</button></div>{error&&<div role="alert" className="login-error">{error}</div>}<button className="login-submit" disabled={busy} type="submit">{busy?'正在登录…':'登录管理端 →'}</button><div className="login-note">默认账号：admin。首次启动后，请从后端 data/admin-password.txt 文件获取本机生成的密码。</div><button className="connection-check" disabled={busy} type="button" onClick={()=>void check()}>重新检查服务连接</button></form><small className="login-footer">仅管理员可修改问题状态 · 登录有效期 8 小时</small></section></div>;
}
createRoot(document.getElementById('root')!).render(<App/>);
