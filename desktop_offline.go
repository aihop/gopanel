//go:build desktop

package main

import (
	"errors"
	"html"
	"net/http"
	"net/url"
	"strings"
)

const desktopConnectionUnavailableMessage = "目标 GoPanel 服务暂时不可用，请稍后重试或打开连接中心检查设置"

type desktopConnectionFailureCopy struct {
	Title       string
	Description string
}

func writeDesktopConnectionError(response http.ResponseWriter, request *http.Request, target *url.URL, err error) {
	if desktopRequestWantsHTML(request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("Cache-Control", "no-store")
		response.WriteHeader(http.StatusBadGateway)
		_, _ = response.Write([]byte(desktopOfflineHTML(target, err)))
		return
	}
	writeDesktopJSON(response, http.StatusBadGateway, errors.New(desktopConnectionUnavailableMessage))
}

func desktopRequestWantsHTML(request *http.Request) bool {
	if request == nil || request.Method != http.MethodGet {
		return false
	}
	if request.URL.Path == "/" || request.URL.Path == "/index.html" {
		return true
	}
	return strings.Contains(strings.ToLower(request.Header.Get("Accept")), "text/html")
}

func desktopConnectionFailureText(err error) desktopConnectionFailureCopy {
	message := ""
	if err != nil {
		message = strings.ToLower(err.Error())
	}
	switch {
	case strings.Contains(message, "connection refused"):
		return desktopConnectionFailureCopy{
			Title:       "GoPanel 服务暂未启动",
			Description: "目标端口没有服务响应。请确认 GoPanel 服务已启动，或检查连接地址和端口是否发生变化。",
		}
	case strings.Contains(message, "timeout"), strings.Contains(message, "deadline exceeded"):
		return desktopConnectionFailureCopy{
			Title:       "连接服务器超时",
			Description: "服务器暂时没有响应。请检查当前网络、防火墙设置，以及远程服务器是否在线。",
		}
	case strings.Contains(message, "no such host"), strings.Contains(message, "server misbehaving"):
		return desktopConnectionFailureCopy{
			Title:       "找不到这台服务器",
			Description: "服务器地址暂时无法解析。请检查地址拼写、网络连接或 DNS 设置。",
		}
	case strings.Contains(message, "tls"), strings.Contains(message, "certificate"):
		return desktopConnectionFailureCopy{
			Title:       "安全连接验证失败",
			Description: "服务器证书或 HTTPS 配置未能通过验证，请检查证书状态和连接地址。",
		}
	default:
		return desktopConnectionFailureCopy{
			Title:       "暂时无法连接到 GoPanel",
			Description: "桌面应用运行正常，但目标服务当前没有响应。你可以立即重试，或前往连接中心检查设置。",
		}
	}
}

func desktopOfflineHTML(target *url.URL, err error) string {
	copy := desktopConnectionFailureText(err)
	address := "当前服务器"
	if target != nil {
		address = target.String()
	}
	detail := "暂无更多错误信息"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		detail = strings.TrimSpace(err.Error())
	}
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + html.EscapeString(copy.Title) + ` · GoPanel</title>
<style>
*{box-sizing:border-box}*:focus-visible{outline:3px solid rgba(129,140,248,.45);outline-offset:3px}
:root{color-scheme:dark;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
html,body{margin:0;min-width:100%;min-height:100%}body{display:grid;min-height:100vh;place-items:center;overflow:auto;background:radial-gradient(circle at 15% 0,#253556 0,#101827 43%,#080d18 100%);color:#e5edf9;padding:28px}
button{font:inherit}.ambient{position:fixed;inset:0;overflow:hidden;pointer-events:none}.ambient::before,.ambient::after{content:"";position:absolute;border-radius:999px;filter:blur(12px);opacity:.16}.ambient::before{top:-150px;right:-80px;width:420px;height:420px;background:#6366f1}.ambient::after{bottom:-180px;left:-100px;width:380px;height:380px;background:#22c55e}
.card{position:relative;width:min(720px,100%);overflow:hidden;border:1px solid rgba(100,116,139,.5);border-radius:28px;background:rgba(15,23,42,.92);box-shadow:0 36px 100px rgba(0,0,0,.42);backdrop-filter:blur(18px)}
.accent{height:4px;background:linear-gradient(90deg,#6366f1,#38bdf8 52%,#22c55e)}.content{padding:44px 46px 38px}.brand{display:flex;align-items:center;gap:11px;color:#94a3b8;font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase}.logo{display:grid;width:34px;height:34px;place-items:center;border-radius:11px;background:linear-gradient(135deg,#6366f1,#22c55e);color:white;font-size:17px;font-weight:850;letter-spacing:0;box-shadow:0 8px 24px rgba(99,102,241,.3)}
.hero{display:grid;grid-template-columns:auto 1fr;gap:24px;align-items:start;margin-top:34px}.signal{position:relative;display:grid;width:72px;height:72px;place-items:center;border:1px solid rgba(248,113,113,.28);border-radius:22px;background:rgba(127,29,29,.18);color:#fca5a5}.signal::before{content:"";position:absolute;inset:10px;border:1px solid rgba(248,113,113,.2);border-radius:15px;animation:pulse 2s ease-out infinite}.signal svg{position:relative;width:34px;height:34px}.eyebrow{display:flex;align-items:center;gap:8px;margin:2px 0 10px;color:#fca5a5;font-size:12px;font-weight:700}.status-dot{width:7px;height:7px;border-radius:50%;background:#f87171;box-shadow:0 0 0 5px rgba(248,113,113,.1)}h1{margin:0;color:#f8fafc;font-size:30px;line-height:1.25;letter-spacing:-.02em}p{margin:12px 0 0;color:#94a3b8;font-size:14px;line-height:1.75}
.target{display:flex;align-items:center;gap:9px;margin-top:24px;padding:12px 14px;border:1px solid #334155;border-radius:13px;background:#0b1220;color:#cbd5e1;font-size:12px}.target svg{flex:none;width:16px;color:#64748b}.target code{min-width:0;overflow:hidden;font-family:"SFMono-Regular",Consolas,monospace;text-overflow:ellipsis;white-space:nowrap}
.actions{display:flex;flex-wrap:wrap;align-items:center;gap:11px;margin-top:28px}.button{display:inline-flex;align-items:center;justify-content:center;gap:8px;min-height:44px;border-radius:12px;padding:0 17px;font-weight:700;cursor:pointer;transition:transform .15s ease,background .15s ease,border-color .15s ease}.button:hover{transform:translateY(-1px)}.button:disabled{cursor:wait;opacity:.65;transform:none}.primary{border:0;background:#6366f1;color:#fff;box-shadow:0 10px 28px rgba(99,102,241,.25)}.primary:hover{background:#7073f4}.secondary{border:1px solid #475569;background:#172033;color:#e2e8f0}.secondary:hover{border-color:#64748b;background:#1e293b}.button svg{width:16px;height:16px}.retry-status{margin-left:auto;color:#64748b;font-size:12px}
.tips{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-top:34px}.tip{padding:14px;border:1px solid rgba(51,65,85,.7);border-radius:14px;background:rgba(17,24,39,.62)}.tip strong{display:block;color:#cbd5e1;font-size:12px}.tip span{display:block;margin-top:5px;color:#64748b;font-size:11px;line-height:1.55}
details{margin-top:22px;border-top:1px solid #273449;padding-top:18px;color:#64748b;font-size:11px}summary{width:max-content;cursor:pointer;user-select:none}details code{display:block;margin-top:12px;overflow:auto;border-radius:10px;background:#080d18;padding:12px;color:#94a3b8;font-family:"SFMono-Regular",Consolas,monospace;line-height:1.5;white-space:pre-wrap;word-break:break-word}
@keyframes pulse{0%{transform:scale(.84);opacity:.7}70%,100%{transform:scale(1.32);opacity:0}}@media(max-width:650px){body{padding:16px}.content{padding:30px 24px}.hero{grid-template-columns:1fr;gap:18px}.signal{width:62px;height:62px}.tips{grid-template-columns:1fr}.retry-status{width:100%;margin-left:0}h1{font-size:25px}}
</style>
</head>
<body>
<div class="ambient"></div>
<main class="card">
  <div class="accent"></div>
  <div class="content">
    <div class="brand"><span class="logo">G</span><span>GoPanel Desktop</span></div>
    <section class="hero">
      <div class="signal"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M5 12.55a11 11 0 0 1 14.08 0M8.53 16.11a6 6 0 0 1 6.95 0M12 20h.01"/><path d="m4 4 16 16"/></svg></div>
      <div><div class="eyebrow"><span class="status-dot"></span>等待服务恢复</div><h1>` + html.EscapeString(copy.Title) + `</h1><p>` + html.EscapeString(copy.Description) + `</p></div>
    </section>
    <div class="target"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><rect x="3" y="4" width="18" height="16" rx="3"/><path d="M7 8h.01M7 12h10M7 16h7"/></svg><code>` + html.EscapeString(address) + `</code></div>
    <div class="actions">
      <button id="retryButton" class="button primary" type="button" onclick="probeConnection()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 12a8 8 0 1 1-2.34-5.66L20 8"/><path d="M20 3v5h-5"/></svg><span>立即重试</span></button>
      <button class="button secondary" type="button" onclick="location.href='/__desktop'"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 15.5A3.5 3.5 0 1 0 12 8a3.5 3.5 0 0 0 0 7.5Z"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.83 2.83-.06-.06A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6l-.04.08V20h-4v-.08A1.7 1.7 0 0 0 9 19.4a1.7 1.7 0 0 0-1.88.34l-.06.06-2.83-2.83.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1H4v-4h.08A1.7 1.7 0 0 0 4.6 9a1.7 1.7 0 0 0-.34-1.88l-.06-.06 2.83-2.83.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6l.04-.08V4h4v.08A1.7 1.7 0 0 0 15 4.6a1.7 1.7 0 0 0 1.88-.34l.06-.06 2.83 2.83-.06.06A1.7 1.7 0 0 0 19.4 9c.11.37.32.7.6 1h.08v4H20a1.7 1.7 0 0 0-.6 1Z"/></svg><span>打开连接中心</span></button>
      <span id="retryStatus" class="retry-status">5 秒后自动检测</span>
    </div>
    <div class="tips"><div class="tip"><strong>确认服务状态</strong><span>检查 GoPanel 服务是否已经启动。</span></div><div class="tip"><strong>核对连接地址</strong><span>确认服务器地址和端口没有变化。</span></div><div class="tip"><strong>检查网络策略</strong><span>远程连接需允许对应端口访问。</span></div></div>
    <details><summary>查看技术详情</summary><code>` + html.EscapeString(detail) + `</code></details>
  </div>
</main>
<script>
const retryButton=document.getElementById('retryButton');const retryStatus=document.getElementById('retryStatus');let remaining=5;let probing=false;
function renderCountdown(){if(!probing)retryStatus.textContent=remaining+' 秒后自动检测'}
async function probeConnection(){if(probing)return;probing=true;retryButton.disabled=true;retryButton.querySelector('span').textContent='正在重试…';retryStatus.textContent='正在检查服务状态';try{const response=await fetch('/health',{cache:'no-store',headers:{Accept:'application/json'}});if(response.ok){retryStatus.textContent='连接已恢复，正在返回…';location.replace('/');return}}catch(_){}probing=false;remaining=5;retryButton.disabled=false;retryButton.querySelector('span').textContent='立即重试';retryStatus.textContent='仍未连接，将继续自动检测'}
setInterval(()=>{if(probing)return;remaining-=1;if(remaining<=0){probeConnection();return}renderCountdown()},1000);
</script>
</body>
</html>`
}
