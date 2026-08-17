//go:build desktop

package main

import "strconv"

func (gateway *desktopGateway) launcherHTML() string {
	defaultAddress := "http://127.0.0.1:5470"
	if discovered := discoverLocalDesktopTarget(gateway.baseDir); discovered != nil {
		defaultAddress = discovered.String()
	}
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>GoPanel 连接中心</title>
<style>
*{box-sizing:border-box;scrollbar-width:none}*::-webkit-scrollbar{display:none;width:0;height:0}
:root{color-scheme:dark;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
html,body{margin:0;width:100%;height:100%;overflow:hidden}body{background:radial-gradient(circle at 15% 0,#253556 0,#101827 43%,#080d18 100%);color:#e5edf9}
button,input{font:inherit}.shell{display:grid;grid-template-rows:auto 1fr;width:100%;height:100%;padding:24px 28px 28px}.header{display:flex;align-items:center;justify-content:space-between;gap:20px;margin-bottom:20px}.brand{display:flex;align-items:center;gap:12px}.logo{display:grid;place-items:center;width:42px;height:42px;border-radius:14px;background:linear-gradient(135deg,#6366f1,#22c55e);font-size:21px;font-weight:800}.brand strong{font-size:20px}.brand span{display:block;margin-top:3px;color:#94a3b8;font-size:12px}.back{border:1px solid #334155;border-radius:11px;background:#172033;color:#cbd5e1;padding:9px 14px;cursor:pointer}.layout{display:grid;grid-template-columns:minmax(290px,380px) minmax(380px,1fr);gap:18px;min-height:0}.panel{min-height:0;border:1px solid #334155;border-radius:22px;background:rgba(15,23,42,.9);box-shadow:0 24px 70px rgba(0,0,0,.28)}.sidebar{display:flex;flex-direction:column;padding:20px}.section-title{margin:0 0 12px;color:#94a3b8;font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase}.current{padding:16px;border:1px solid #334155;border-radius:16px;background:#111c30}.current-top{display:flex;align-items:center;justify-content:space-between;gap:12px}.current-name{min-width:0;font-weight:700}.current-address{margin-top:6px;overflow:hidden;color:#94a3b8;font-size:12px;text-overflow:ellipsis;white-space:nowrap}.badge{display:flex;align-items:center;gap:6px;flex:none;padding:5px 8px;border-radius:999px;background:#1e293b;color:#cbd5e1;font-size:11px}.dot{width:7px;height:7px;border-radius:50%;background:#f59e0b}.dot.online{background:#22c55e}.dot.offline{background:#ef4444}.builtin{display:flex;align-items:center;justify-content:space-between;gap:14px;margin:14px 0 20px;padding:13px 14px;border:1px solid #334155;border-radius:14px;background:#111827}.builtin strong{font-size:13px}.builtin span{display:block;margin-top:3px;color:#64748b;font-size:11px}.small-button{border:1px solid #475569;border-radius:9px;background:#1e293b;color:#e2e8f0;padding:8px 11px;font-size:12px;cursor:pointer}.recent{display:flex;min-height:0;flex:1;flex-direction:column}.server-list{display:grid;gap:8px;overflow:auto}.server{display:grid;grid-template-columns:1fr auto;gap:8px;align-items:center;padding:11px 12px;border:1px solid transparent;border-radius:13px;background:#111827;cursor:pointer}.server:hover,.server.selected{border-color:#6366f1;background:#17203a}.server-name{overflow:hidden;font-size:13px;font-weight:650;text-overflow:ellipsis;white-space:nowrap}.server-url{overflow:hidden;margin-top:3px;color:#64748b;font-size:11px;text-overflow:ellipsis;white-space:nowrap}.delete{border:0;background:transparent;color:#64748b;padding:7px;cursor:pointer}.delete:hover{color:#f87171}.empty{padding:26px 12px;border:1px dashed #334155;border-radius:14px;color:#64748b;text-align:center;font-size:12px}.editor{display:flex;flex-direction:column;padding:28px}.editor-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px}.editor h1{margin:0 0 8px;font-size:25px}.editor p{margin:0;color:#94a3b8;font-size:13px;line-height:1.6}.new{border:1px solid #475569;border-radius:10px;background:transparent;color:#cbd5e1;padding:9px 12px;cursor:pointer}.form{display:grid;gap:16px;margin-top:28px}.field{display:grid;gap:7px}.field label{color:#cbd5e1;font-size:12px;font-weight:650}.field small{color:#64748b;font-size:11px}.saved-entry{display:flex;align-items:center;gap:8px;color:#94a3b8;font-size:11px}.saved-entry input{width:auto}input{width:100%;min-width:0;border:1px solid #475569;border-radius:12px;background:#0b1220;color:#f8fafc;padding:13px 15px;outline:none}input:focus{border-color:#818cf8;box-shadow:0 0 0 3px rgba(99,102,241,.18)}.actions{display:flex;align-items:center;gap:12px;margin-top:24px}.primary{border:0;border-radius:12px;background:#6366f1;color:#fff;padding:12px 18px;font-weight:700;cursor:pointer}.primary:disabled,.small-button:disabled{cursor:wait;opacity:.55}.status{min-height:22px;color:#fca5a5;font-size:12px}.status.success{color:#86efac}.notice{margin-top:auto;padding:14px 16px;border-radius:13px;background:#111c30;color:#94a3b8;font-size:12px;line-height:1.6}
@media(max-width:760px){.shell{padding:16px}.layout{grid-template-columns:1fr;overflow:auto}.sidebar,.editor{min-height:420px}.header{margin-bottom:14px}}
</style>
</head>
<body>
<main class="shell">
  <header class="header"><div class="brand"><div class="logo">G</div><div><strong>服务器连接中心</strong><span>GoPanel Desktop</span></div></div><button class="back" onclick="location.href='/'">返回管理台</button></header>
  <div class="layout">
    <section class="panel sidebar">
      <h2 class="section-title">当前连接</h2>
      <div class="current"><div class="current-top"><div id="currentName" class="current-name">正在检测…</div><div class="badge"><span id="currentDot" class="dot"></span><span id="currentStatus">检测中</span></div></div><div id="currentAddress" class="current-address">—</div></div>
      <div class="builtin"><div><strong>本机内置服务</strong><span>使用 ~/.gopanel 数据目录</span></div><button id="builtinButton" class="small-button" onclick="startBuiltin()">切换</button></div>
      <div class="recent"><h2 class="section-title">最近服务器</h2><div id="serverList" class="server-list"><div class="empty">正在加载服务器…</div></div></div>
    </section>
    <section class="panel editor">
      <div class="editor-head"><div><h1 id="editorTitle">添加服务器</h1><p>切换前会验证服务身份和专属安全入口，避免连接到错误目标。</p></div><button class="new" onclick="newServer()">添加服务器</button></div>
      <div class="form">
        <div class="field"><label for="name">服务器名称</label><input id="name" maxlength="80" placeholder="例如：生产服务器"><small>仅保存在当前电脑，用于识别不同环境。</small></div>
        <div class="field"><label for="address">Web 服务地址</label><input id="address" placeholder="https://panel.example.com:15470"></div>
        <div class="field"><label for="entrance">专属安全入口</label><input id="entrance" maxlength="255" placeholder="可选，例如：your-entrance"><small id="entranceHint">也可以直接在服务地址中粘贴完整的专属入口 URL。</small><label id="clearEntranceLabel" class="saved-entry" hidden><input id="clearEntrance" type="checkbox"> 清除已保存的安全入口</label></div>
      </div>
      <div class="actions"><button id="connectButton" class="primary" onclick="connectRemote()">验证并切换</button><div id="status" class="status"></div></div>
      <div class="notice">切换服务器后会清除上一台服务器的登录态和安全入口缓存，但保留主题与语言设置。若内置服务正在运行，系统会保存目标并提示重启桌面应用。</div>
    </section>
  </div>
</main>
<script>
const defaultAddress=` + strconv.Quote(defaultAddress) + `;
const els={name:document.getElementById('name'),address:document.getElementById('address'),entrance:document.getElementById('entrance'),entranceHint:document.getElementById('entranceHint'),clearEntrance:document.getElementById('clearEntrance'),clearEntranceLabel:document.getElementById('clearEntranceLabel'),status:document.getElementById('status'),list:document.getElementById('serverList'),connect:document.getElementById('connectButton'),builtin:document.getElementById('builtinButton')};
let state={current:{},servers:[]};let selected='';
function setStatus(text,success=false){els.status.textContent=text;els.status.className='status'+(success?' success':'')}
function setLoading(value){els.connect.disabled=value;els.builtin.disabled=value;els.connect.textContent=value?'正在验证…':'验证并切换'}
function escapeText(value){return String(value||'')}
function renderState(){const current=state.current||{};document.getElementById('currentName').textContent=current.name||'未连接';document.getElementById('currentAddress').textContent=current.url||'尚未选择服务器';document.getElementById('currentStatus').textContent=current.online?'在线':(current.url?'离线':'未连接');document.getElementById('currentDot').className='dot '+(current.online?'online':'offline');els.list.replaceChildren();if(!state.servers||state.servers.length===0){const empty=document.createElement('div');empty.className='empty';empty.textContent='暂无最近服务器，请在右侧添加';els.list.appendChild(empty);return}state.servers.forEach(server=>{const item=document.createElement('div');item.className='server'+(selected===server.url?' selected':'');item.onclick=()=>selectServer(server);const info=document.createElement('div');const name=document.createElement('div');name.className='server-name';name.textContent=escapeText(server.name);const url=document.createElement('div');url.className='server-url';url.textContent=escapeText(server.url);info.append(name,url);const remove=document.createElement('button');remove.className='delete';remove.title='删除此记录';remove.textContent='×';remove.onclick=event=>{event.stopPropagation();deleteServer(server)};item.append(info,remove);els.list.appendChild(item)})}
function selectServer(server){selected=server.url;els.name.value=server.name||'';els.address.value=server.url||'';els.entrance.value='';els.entrance.placeholder=server.hasEntrance?'已保存，留空继续使用':'可选，例如：your-entrance';els.entranceHint.textContent=server.hasEntrance?'安全入口已保存在当前电脑，输入新值可覆盖。':'也可以直接在服务地址中粘贴完整的专属入口 URL。';els.clearEntrance.checked=false;els.clearEntranceLabel.hidden=!server.hasEntrance;document.getElementById('editorTitle').textContent='编辑服务器';setStatus('');renderState()}
function newServer(){selected='';els.name.value='';els.address.value='';els.entrance.value='';els.entrance.placeholder='可选，例如：your-entrance';els.entranceHint.textContent='也可以直接在服务地址中粘贴完整的专属入口 URL。';els.clearEntrance.checked=false;els.clearEntranceLabel.hidden=true;document.getElementById('editorTitle').textContent='添加服务器';setStatus('');renderState();els.name.focus()}
function clearServerSession(){localStorage.removeItem('__gopanel__auth');localStorage.removeItem('GlobalState');localStorage.removeItem('__gopanel__GlobalState');sessionStorage.clear()}
async function request(path,body){const response=await fetch(path,{method:body?'POST':'GET',headers:body?{'Content-Type':'application/json'}:undefined,body:body?JSON.stringify(body):undefined,cache:'no-store'});const data=await response.json();if(!response.ok&&!data.restart)throw new Error(data.error||'操作失败');return data}
async function loadState(){try{const result=await request('/__desktop/state');state=result.data||{current:{},servers:[]};renderState();if(state.servers.length){const current=state.servers.find(server=>server.url===state.current.url);selectServer(current||state.servers[0])}else{els.address.value=defaultAddress}}catch(error){els.list.innerHTML='<div class="empty">服务器列表加载失败</div>';setStatus(error.message||'加载失败')}}
async function connectRemote(){if(!els.address.value.trim()){setStatus('请输入服务器地址');els.address.focus();return}setLoading(true);setStatus('正在验证服务器身份和安全入口…');try{const result=await request('/__desktop/connect',{name:els.name.value,url:els.address.value,entrance:els.entrance.value,clearEntrance:els.clearEntrance.checked});if(result.restart){setStatus(result.error);return}clearServerSession();setStatus('切换成功，正在进入管理台…',true);location.href='/'}catch(error){setStatus(error.message||'连接失败')}finally{setLoading(false)}}
async function startBuiltin(){setLoading(true);setStatus('正在启动本机内置服务…');try{await request('/__desktop/builtin',{});clearServerSession();setStatus('内置服务已启动，正在进入管理台…',true);location.href='/'}catch(error){setStatus(error.message||'内置服务启动失败')}finally{setLoading(false)}}
async function deleteServer(server){if(server.url===state.current.url){setStatus('当前连接不能删除，请先切换到其他服务器');return}if(!confirm('从最近服务器中删除“'+server.name+'”？'))return;try{await request('/__desktop/delete',{url:server.url});state.servers=state.servers.filter(item=>item.url!==server.url);if(selected===server.url)newServer();else renderState();setStatus('服务器记录已删除',true)}catch(error){setStatus(error.message||'删除失败')}}
for(const input of [els.name,els.address,els.entrance])input.addEventListener('keydown',event=>{if(event.key==='Enter')connectRemote()});loadState();
</script>
</body>
</html>`
}
