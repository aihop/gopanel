//go:build desktop

package main

import "html/template"

func (gateway *desktopGateway) launcherHTML() string {
	gateway.RLock()
	config := gateway.config
	gateway.RUnlock()
	address := config.URL
	if address == "" {
		address = "http://127.0.0.1:5470"
		if discovered := discoverLocalDesktopTarget(gateway.baseDir); discovered != nil {
			address = discovered.String()
		}
	}
	entrance := config.Entrance
	if entrance == "" {
		if target, err := normalizeDesktopTarget(address); err == nil {
			entrance = discoverLocalDesktopEntrance(gateway.baseDir, target)
		}
	}
	return `<!doctype html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>连接 GoPanel</title><style>
*{box-sizing:border-box;scrollbar-width:none}*::-webkit-scrollbar{display:none;width:0;height:0}html,body{margin:0;width:100%;height:100%;overflow:hidden}body{display:grid;place-items:center;background:radial-gradient(circle at top,#24324d 0,#0f172a 48%,#080d18 100%);color:#e5edf9;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}.card{width:min(680px,calc(100vw - 48px));padding:38px;border:1px solid #334155;border-radius:24px;background:rgba(15,23,42,.94);box-shadow:0 30px 80px rgba(0,0,0,.38)}.brand{display:flex;align-items:center;gap:12px;margin-bottom:28px}.logo{display:grid;place-items:center;width:44px;height:44px;border-radius:14px;background:linear-gradient(135deg,#6366f1,#22c55e);font-size:22px;font-weight:800}.brand strong{font-size:22px}.brand span{display:block;margin-top:4px;color:#94a3b8;font-size:13px}h1{margin:0 0 8px;font-size:28px}p{margin:0 0 24px;color:#94a3b8;line-height:1.65}.fields{display:grid;gap:14px}.field{display:grid;gap:7px}.field label{color:#cbd5e1;font-size:13px;font-weight:600}.field-row{display:flex;gap:10px}input{width:100%;min-width:0;border:1px solid #475569;border-radius:12px;background:#0b1220;color:#f8fafc;padding:13px 15px;font-size:15px;outline:none}input:focus{border-color:#818cf8;box-shadow:0 0 0 3px rgba(99,102,241,.2)}button{border:0;border-radius:12px;padding:0 18px;font-size:14px;font-weight:650;cursor:pointer}.primary{min-width:112px;background:#6366f1;color:white}.secondary{width:100%;height:46px;margin-top:12px;background:#1e293b;color:#e2e8f0;border:1px solid #334155}.divider{display:flex;align-items:center;gap:12px;margin:22px 0;color:#64748b;font-size:12px}.divider:before,.divider:after{content:"";height:1px;flex:1;background:#334155}.status{min-height:22px;margin-top:12px;color:#fca5a5;font-size:13px}.hint{margin-top:22px;padding:14px 16px;border-radius:12px;background:#111c30;color:#94a3b8;font-size:13px;line-height:1.55}
</style></head><body><main class="card"><div class="brand"><div class="logo">G</div><div><strong>GoPanel</strong><span>桌面控制台</span></div></div><h1>连接你的 GoPanel</h1><p>连接已经运行的 Web 服务，桌面端只作为客户端；也可以在本机启动内置服务。</p><div class="fields"><div class="field"><label for="address">Web 服务地址</label><input id="address" value="` + template.HTMLEscapeString(address) + `" placeholder="http://服务器IP:端口"></div><div class="field"><label for="entrance">专属安全入口</label><div class="field-row"><input id="entrance" value="` + template.HTMLEscapeString(entrance) + `" placeholder="例如：your-entrance"><button class="primary" onclick="connectRemote()">保存并连接</button></div></div></div><div id="status" class="status"></div><div class="divider">或者</div><button class="secondary" onclick="startBuiltin()">启动本机内置服务</button><div class="hint">可直接粘贴完整的专属安全入口 URL；也可分别填写服务地址和入口。远程模式不会打开本地数据库，内置模式默认使用 <b>~/.gopanel</b>。</div></main><script>
const status=document.getElementById('status');async function submit(path,body){status.textContent='正在连接…';try{const response=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json'},body:body?JSON.stringify(body):'{}'});const data=await response.json();if(data.restart){status.textContent=data.error;return}if(!response.ok)throw new Error(data.error||'连接失败');location.href='/';}catch(error){status.textContent=error.message||'连接失败';}}function connectRemote(){submit('/__desktop/connect',{url:document.getElementById('address').value,entrance:document.getElementById('entrance').value})}function startBuiltin(){submit('/__desktop/builtin')}for(const id of ['address','entrance'])document.getElementById(id).addEventListener('keydown',event=>{if(event.key==='Enter')connectRemote()});
</script></body></html>`
}
