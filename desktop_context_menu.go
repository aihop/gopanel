//go:build desktop

package main

func desktopContextMenuHTML() string {
	return `<style>
#gopanel-desktop-context-menu{position:fixed;z-index:2147483647;display:none;min-width:190px;padding:6px;border:1px solid rgba(148,163,184,.3);border-radius:10px;background:rgba(15,23,42,.98);box-shadow:0 14px 35px rgba(0,0,0,.32);color:#e2e8f0;font:13px -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;backdrop-filter:blur(12px)}
#gopanel-desktop-context-menu[data-open="true"]{display:block}
#gopanel-desktop-context-menu button{display:flex;align-items:center;width:100%;gap:10px;padding:8px 10px;border:0;border-radius:6px;background:transparent;color:inherit;font:inherit;text-align:left;cursor:pointer}
#gopanel-desktop-context-menu button:hover:not(:disabled){background:rgba(99,102,241,.28)}
#gopanel-desktop-context-menu button:disabled{color:#64748b;cursor:default}
#gopanel-desktop-context-menu .gopanel-context-menu-separator{height:1px;margin:5px 4px;background:rgba(148,163,184,.2)}
#gopanel-desktop-context-menu kbd{margin-left:auto;color:#94a3b8;font-size:11px}
</style>
<script>
(()=>{
  const init=()=>{
    if(window.__GOPANEL_DESKTOP_CONTEXT_MENU__)return;
    window.__GOPANEL_DESKTOP_CONTEXT_MENU__=true;
    const menu=document.createElement('div');
    menu.id='gopanel-desktop-context-menu';
    menu.setAttribute('role','menu');
    menu.innerHTML='<button type="button" data-action="reload" role="menuitem"><span data-label="reload">刷新页面</span><kbd>⌘/Ctrl+R</kbd></button><button type="button" data-action="back" role="menuitem"><span data-label="back">后退</span><kbd>Alt+←</kbd></button><button type="button" data-action="forward" role="menuitem"><span data-label="forward">前进</span><kbd>Alt+→</kbd></button><button type="button" data-action="home" role="menuitem"><span data-label="home">返回首页</span></button><div class="gopanel-context-menu-separator" role="separator"></div><button type="button" data-action="copy" role="menuitem"><span data-label="copy">复制当前地址</span></button><button type="button" data-action="connection" role="menuitem"><span data-label="connection">打开连接中心</span></button>';
    document.body.appendChild(menu);
    const labels={
      zh:{reload:'刷新页面',back:'后退',forward:'前进',home:'返回首页',copy:'复制当前地址',connection:'打开连接中心'},
      en:{reload:'Refresh page',back:'Back',forward:'Forward',home:'Home',copy:'Copy current address',connection:'Open connection center'}
    };
    const locale=(localStorage.getItem('lang')||'').toLowerCase().startsWith('en')?'en':'zh';
    menu.querySelectorAll('[data-label]').forEach(item=>{item.textContent=labels[locale][item.dataset.label]||item.textContent});
    const canGoBack=()=>window.history.length>1;
    const close=()=>menu.removeAttribute('data-open');
    const open=(x,y)=>{
      const back=menu.querySelector('[data-action="back"]');
      back.disabled=!canGoBack();
      menu.setAttribute('data-open','true');
      const rect=menu.getBoundingClientRect();
      menu.style.left=Math.max(6,Math.min(x,window.innerWidth-rect.width-6))+'px';
      menu.style.top=Math.max(6,Math.min(y,window.innerHeight-rect.height-6))+'px';
    };
    const canUseNativeMenu=target=>{
      if(!(target instanceof Element))return true;
      return Boolean(target.closest('input,textarea,select,[contenteditable="true"],.monaco-editor,.xterm,[data-desktop-native-context-menu]')||window.getSelection()?.toString());
    };
    document.addEventListener('contextmenu',event=>{
      if(canUseNativeMenu(event.target))return;
      event.preventDefault();
      open(event.clientX,event.clientY);
    },true);
    document.addEventListener('pointerdown',event=>{if(!menu.contains(event.target))close()},true);
    document.addEventListener('keydown',event=>{if(event.key==='Escape')close()});
    document.addEventListener('scroll',close,true);
    window.addEventListener('resize',close);
    window.addEventListener('blur',close);
    menu.addEventListener('click',async event=>{
      const button=event.target.closest('button');
      if(!button||button.disabled)return;
      close();
      switch(button.dataset.action){
        case 'reload':window.location.reload();break;
        case 'back':window.history.back();break;
        case 'forward':window.history.forward();break;
        case 'home':window.location.href='/';break;
        case 'connection':window.location.href='/__desktop';break;
        case 'copy':
          try{await navigator.clipboard.writeText(window.location.href)}catch{
            const input=document.createElement('textarea');input.value=window.location.href;input.style.position='fixed';input.style.opacity='0';document.body.appendChild(input);input.select();document.execCommand('copy');input.remove();
          }
          break;
      }
    });
  };
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',init,{once:true});else init();
})();
</script>`
}
