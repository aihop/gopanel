import{k as se,m as l,i as we,s as dt,F as bt,c as J,r as $,w as ee,o as ct,p as ft,t as E,y as pt,l as ut,v as vt,A as ht,B as gt,n as te}from"./editor-vue-Ca7D1xBp.js";import{p as xt,a0 as mt,a2 as yt,a1 as wt,bB as Ct,N as St,av as Rt,e as r,f as n,h as p,g as m,A as zt,X as ae,t as he,V as re,cD as $t,u as Tt,v as Ce,$ as ge,aD as Pt,cE as _t,w as j,F as K,x as Wt,aJ as Bt,H as q}from"./naive-ui-data-table-aZaSuRvo.js";import{j as oe}from"./util-vendor-CEM9UkdS.js";import{ai as Lt}from"./index-D3YKcHFq.js";const le=xt("n-tabs"),Se={tab:[String,Number,Object,Function],name:{type:[String,Number],required:!0},disabled:Boolean,displayDirective:{type:String,default:"if"},closable:{type:Boolean,default:void 0},tabProps:Object,label:[String,Number,Object,Function]},It=se({__TAB_PANE__:!0,name:"TabPane",alias:["TabPanel"],props:Se,slots:Object,setup(e){const s=we(le,null);return s||mt("tab-pane","`n-tab-pane` must be placed inside `n-tabs`."),{style:s.paneStyleRef,class:s.paneClassRef,mergedClsPrefix:s.mergedClsPrefixRef}},render(){return l("div",{class:[`${this.mergedClsPrefix}-tab-pane`,this.class],style:this.style},this.$slots)}}),At=Object.assign({internalLeftPadded:Boolean,internalAddable:Boolean,internalCreatedByPane:Boolean},Rt(Se,["displayDirective"])),ie=se({__TAB__:!0,inheritAttrs:!1,name:"Tab",props:At,setup(e){const{mergedClsPrefixRef:s,valueRef:b,typeRef:w,closableRef:v,tabStyleRef:_,addTabStyleRef:u,tabClassRef:y,addTabClassRef:C,tabChangeIdRef:h,onBeforeLeaveRef:c,triggerRef:k,handleAdd:L,activateTab:g,handleClose:S}=we(le);return{trigger:k,mergedClosable:J(()=>{if(e.internalAddable)return!1;const{closable:x}=e;return x===void 0?v.value:x}),style:_,addStyle:u,tabClass:y,addTabClass:C,clsPrefix:s,value:b,type:w,handleClose(x){x.stopPropagation(),!e.disabled&&S(e.name)},activateTab(){if(e.disabled)return;if(e.internalAddable){L();return}const{name:x}=e,T=++h.id;if(x!==b.value){const{value:A}=c;A?Promise.resolve(A(e.name,b.value)).then(z=>{z&&h.id===T&&g(x)}):g(x)}}}},render(){const{internalAddable:e,clsPrefix:s,name:b,disabled:w,label:v,tab:_,value:u,mergedClosable:y,trigger:C,$slots:{default:h}}=this,c=v??_;return l("div",{class:`${s}-tabs-tab-wrapper`},this.internalLeftPadded?l("div",{class:`${s}-tabs-tab-pad`}):null,l("div",Object.assign({key:b,"data-name":b,"data-disabled":w?!0:void 0},dt({class:[`${s}-tabs-tab`,u===b&&`${s}-tabs-tab--active`,w&&`${s}-tabs-tab--disabled`,y&&`${s}-tabs-tab--closable`,e&&`${s}-tabs-tab--addable`,e?this.addTabClass:this.tabClass],onClick:C==="click"?this.activateTab:void 0,onMouseenter:C==="hover"?this.activateTab:void 0,style:e?this.addStyle:this.style},this.internalCreatedByPane?this.tabProps||{}:this.$attrs)),l("span",{class:`${s}-tabs-tab__label`},e?l(bt,null,l("div",{class:`${s}-tabs-tab__height-placeholder`}," "),l(wt,{clsPrefix:s},{default:()=>l(Ct,null)})):h?h():typeof c=="object"?c:yt(c??b)),y&&this.type==="card"?l(St,{clsPrefix:s,class:`${s}-tabs-tab__close`,onClick:this.handleClose,disabled:w}):null))}}),Et=r("tabs",`
 box-sizing: border-box;
 width: 100%;
 display: flex;
 flex-direction: column;
 transition:
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
`,[n("segment-type",[r("tabs-rail",[p("&.transition-disabled",[r("tabs-capsule",`
 transition: none;
 `)])])]),n("top",[r("tab-pane",`
 padding: var(--n-pane-padding-top) var(--n-pane-padding-right) var(--n-pane-padding-bottom) var(--n-pane-padding-left);
 `)]),n("left",[r("tab-pane",`
 padding: var(--n-pane-padding-right) var(--n-pane-padding-bottom) var(--n-pane-padding-left) var(--n-pane-padding-top);
 `)]),n("left, right",`
 flex-direction: row;
 `,[r("tabs-bar",`
 width: 2px;
 right: 0;
 transition:
 top .2s var(--n-bezier),
 max-height .2s var(--n-bezier),
 background-color .3s var(--n-bezier);
 `),r("tabs-tab",`
 padding: var(--n-tab-padding-vertical); 
 `)]),n("right",`
 flex-direction: row-reverse;
 `,[r("tab-pane",`
 padding: var(--n-pane-padding-left) var(--n-pane-padding-top) var(--n-pane-padding-right) var(--n-pane-padding-bottom);
 `),r("tabs-bar",`
 left: 0;
 `)]),n("bottom",`
 flex-direction: column-reverse;
 justify-content: flex-end;
 `,[r("tab-pane",`
 padding: var(--n-pane-padding-bottom) var(--n-pane-padding-right) var(--n-pane-padding-top) var(--n-pane-padding-left);
 `),r("tabs-bar",`
 top: 0;
 `)]),r("tabs-rail",`
 position: relative;
 padding: 3px;
 border-radius: var(--n-tab-border-radius);
 width: 100%;
 background-color: var(--n-color-segment);
 transition: background-color .3s var(--n-bezier);
 display: flex;
 align-items: center;
 `,[r("tabs-capsule",`
 border-radius: var(--n-tab-border-radius);
 position: absolute;
 pointer-events: none;
 background-color: var(--n-tab-color-segment);
 box-shadow: 0 1px 3px 0 rgba(0, 0, 0, .08);
 transition: transform 0.3s var(--n-bezier);
 `),r("tabs-tab-wrapper",`
 flex-basis: 0;
 flex-grow: 1;
 display: flex;
 align-items: center;
 justify-content: center;
 `,[r("tabs-tab",`
 overflow: hidden;
 border-radius: var(--n-tab-border-radius);
 width: 100%;
 display: flex;
 align-items: center;
 justify-content: center;
 `,[n("active",`
 font-weight: var(--n-font-weight-strong);
 color: var(--n-tab-text-color-active);
 `),p("&:hover",`
 color: var(--n-tab-text-color-hover);
 `)])])]),n("flex",[r("tabs-nav",`
 width: 100%;
 position: relative;
 `,[r("tabs-wrapper",`
 width: 100%;
 `,[r("tabs-tab",`
 margin-right: 0;
 `)])])]),r("tabs-nav",`
 box-sizing: border-box;
 line-height: 1.5;
 display: flex;
 transition: border-color .3s var(--n-bezier);
 `,[m("prefix, suffix",`
 display: flex;
 align-items: center;
 `),m("prefix","padding-right: 16px;"),m("suffix","padding-left: 16px;")]),n("top, bottom",[r("tabs-nav-scroll-wrapper",[p("&::before",`
 top: 0;
 bottom: 0;
 left: 0;
 width: 20px;
 `),p("&::after",`
 top: 0;
 bottom: 0;
 right: 0;
 width: 20px;
 `),n("shadow-start",[p("&::before",`
 box-shadow: inset 10px 0 8px -8px rgba(0, 0, 0, .12);
 `)]),n("shadow-end",[p("&::after",`
 box-shadow: inset -10px 0 8px -8px rgba(0, 0, 0, .12);
 `)])])]),n("left, right",[r("tabs-nav-scroll-content",`
 flex-direction: column;
 `),r("tabs-nav-scroll-wrapper",[p("&::before",`
 top: 0;
 left: 0;
 right: 0;
 height: 20px;
 `),p("&::after",`
 bottom: 0;
 left: 0;
 right: 0;
 height: 20px;
 `),n("shadow-start",[p("&::before",`
 box-shadow: inset 0 10px 8px -8px rgba(0, 0, 0, .12);
 `)]),n("shadow-end",[p("&::after",`
 box-shadow: inset 0 -10px 8px -8px rgba(0, 0, 0, .12);
 `)])])]),r("tabs-nav-scroll-wrapper",`
 flex: 1;
 position: relative;
 overflow: hidden;
 `,[r("tabs-nav-y-scroll",`
 height: 100%;
 width: 100%;
 overflow-y: auto; 
 scrollbar-width: none;
 `,[p("&::-webkit-scrollbar, &::-webkit-scrollbar-track-piece, &::-webkit-scrollbar-thumb",`
 width: 0;
 height: 0;
 display: none;
 `)]),p("&::before, &::after",`
 transition: box-shadow .3s var(--n-bezier);
 pointer-events: none;
 content: "";
 position: absolute;
 z-index: 1;
 `)]),r("tabs-nav-scroll-content",`
 display: flex;
 position: relative;
 min-width: 100%;
 min-height: 100%;
 width: fit-content;
 box-sizing: border-box;
 `),r("tabs-wrapper",`
 display: inline-flex;
 flex-wrap: nowrap;
 position: relative;
 `),r("tabs-tab-wrapper",`
 display: flex;
 flex-wrap: nowrap;
 flex-shrink: 0;
 flex-grow: 0;
 `),r("tabs-tab",`
 cursor: pointer;
 white-space: nowrap;
 flex-wrap: nowrap;
 display: inline-flex;
 align-items: center;
 color: var(--n-tab-text-color);
 font-size: var(--n-tab-font-size);
 background-clip: padding-box;
 padding: var(--n-tab-padding);
 transition:
 box-shadow .3s var(--n-bezier),
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[n("disabled",{cursor:"not-allowed"}),m("close",`
 margin-left: 6px;
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `),m("label",`
 display: flex;
 align-items: center;
 z-index: 1;
 `)]),r("tabs-bar",`
 position: absolute;
 bottom: 0;
 height: 2px;
 border-radius: 1px;
 background-color: var(--n-bar-color);
 transition:
 left .2s var(--n-bezier),
 max-width .2s var(--n-bezier),
 opacity .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 `,[p("&.transition-disabled",`
 transition: none;
 `),n("disabled",`
 background-color: var(--n-tab-text-color-disabled)
 `)]),r("tabs-pane-wrapper",`
 position: relative;
 overflow: hidden;
 transition: max-height .2s var(--n-bezier);
 `),r("tab-pane",`
 color: var(--n-pane-text-color);
 width: 100%;
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 opacity .2s var(--n-bezier);
 left: 0;
 right: 0;
 top: 0;
 `,[p("&.next-transition-leave-active, &.prev-transition-leave-active, &.next-transition-enter-active, &.prev-transition-enter-active",`
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 transform .2s var(--n-bezier),
 opacity .2s var(--n-bezier);
 `),p("&.next-transition-leave-active, &.prev-transition-leave-active",`
 position: absolute;
 `),p("&.next-transition-enter-from, &.prev-transition-leave-to",`
 transform: translateX(32px);
 opacity: 0;
 `),p("&.next-transition-leave-to, &.prev-transition-enter-from",`
 transform: translateX(-32px);
 opacity: 0;
 `),p("&.next-transition-leave-from, &.next-transition-enter-to, &.prev-transition-leave-from, &.prev-transition-enter-to",`
 transform: translateX(0);
 opacity: 1;
 `)]),r("tabs-tab-pad",`
 box-sizing: border-box;
 width: var(--n-tab-gap);
 flex-grow: 0;
 flex-shrink: 0;
 `),n("line-type, bar-type",[r("tabs-tab",`
 font-weight: var(--n-tab-font-weight);
 box-sizing: border-box;
 vertical-align: bottom;
 `,[p("&:hover",{color:"var(--n-tab-text-color-hover)"}),n("active",`
 color: var(--n-tab-text-color-active);
 font-weight: var(--n-tab-font-weight-active);
 `),n("disabled",{color:"var(--n-tab-text-color-disabled)"})])]),r("tabs-nav",[n("line-type",[n("top",[m("prefix, suffix",`
 border-bottom: 1px solid var(--n-tab-border-color);
 `),r("tabs-nav-scroll-content",`
 border-bottom: 1px solid var(--n-tab-border-color);
 `),r("tabs-bar",`
 bottom: -1px;
 `)]),n("left",[m("prefix, suffix",`
 border-right: 1px solid var(--n-tab-border-color);
 `),r("tabs-nav-scroll-content",`
 border-right: 1px solid var(--n-tab-border-color);
 `),r("tabs-bar",`
 right: -1px;
 `)]),n("right",[m("prefix, suffix",`
 border-left: 1px solid var(--n-tab-border-color);
 `),r("tabs-nav-scroll-content",`
 border-left: 1px solid var(--n-tab-border-color);
 `),r("tabs-bar",`
 left: -1px;
 `)]),n("bottom",[m("prefix, suffix",`
 border-top: 1px solid var(--n-tab-border-color);
 `),r("tabs-nav-scroll-content",`
 border-top: 1px solid var(--n-tab-border-color);
 `),r("tabs-bar",`
 top: -1px;
 `)]),m("prefix, suffix",`
 transition: border-color .3s var(--n-bezier);
 `),r("tabs-nav-scroll-content",`
 transition: border-color .3s var(--n-bezier);
 `),r("tabs-bar",`
 border-radius: 0;
 `)]),n("card-type",[m("prefix, suffix",`
 transition: border-color .3s var(--n-bezier);
 `),r("tabs-pad",`
 flex-grow: 1;
 transition: border-color .3s var(--n-bezier);
 `),r("tabs-tab-pad",`
 transition: border-color .3s var(--n-bezier);
 `),r("tabs-tab",`
 font-weight: var(--n-tab-font-weight);
 border: 1px solid var(--n-tab-border-color);
 background-color: var(--n-tab-color);
 box-sizing: border-box;
 position: relative;
 vertical-align: bottom;
 display: flex;
 justify-content: space-between;
 font-size: var(--n-tab-font-size);
 color: var(--n-tab-text-color);
 `,[n("addable",`
 padding-left: 8px;
 padding-right: 8px;
 font-size: 16px;
 justify-content: center;
 `,[m("height-placeholder",`
 width: 0;
 font-size: var(--n-tab-font-size);
 `),zt("disabled",[p("&:hover",`
 color: var(--n-tab-text-color-hover);
 `)])]),n("closable","padding-right: 8px;"),n("active",`
 background-color: #0000;
 font-weight: var(--n-tab-font-weight-active);
 color: var(--n-tab-text-color-active);
 `),n("disabled","color: var(--n-tab-text-color-disabled);")])]),n("left, right",`
 flex-direction: column; 
 `,[m("prefix, suffix",`
 padding: var(--n-tab-padding-vertical);
 `),r("tabs-wrapper",`
 flex-direction: column;
 `),r("tabs-tab-wrapper",`
 flex-direction: column;
 `,[r("tabs-tab-pad",`
 height: var(--n-tab-gap-vertical);
 width: 100%;
 `)])]),n("top",[n("card-type",[r("tabs-scroll-padding","border-bottom: 1px solid var(--n-tab-border-color);"),m("prefix, suffix",`
 border-bottom: 1px solid var(--n-tab-border-color);
 `),r("tabs-tab",`
 border-top-left-radius: var(--n-tab-border-radius);
 border-top-right-radius: var(--n-tab-border-radius);
 `,[n("active",`
 border-bottom: 1px solid #0000;
 `)]),r("tabs-tab-pad",`
 border-bottom: 1px solid var(--n-tab-border-color);
 `),r("tabs-pad",`
 border-bottom: 1px solid var(--n-tab-border-color);
 `)])]),n("left",[n("card-type",[r("tabs-scroll-padding","border-right: 1px solid var(--n-tab-border-color);"),m("prefix, suffix",`
 border-right: 1px solid var(--n-tab-border-color);
 `),r("tabs-tab",`
 border-top-left-radius: var(--n-tab-border-radius);
 border-bottom-left-radius: var(--n-tab-border-radius);
 `,[n("active",`
 border-right: 1px solid #0000;
 `)]),r("tabs-tab-pad",`
 border-right: 1px solid var(--n-tab-border-color);
 `),r("tabs-pad",`
 border-right: 1px solid var(--n-tab-border-color);
 `)])]),n("right",[n("card-type",[r("tabs-scroll-padding","border-left: 1px solid var(--n-tab-border-color);"),m("prefix, suffix",`
 border-left: 1px solid var(--n-tab-border-color);
 `),r("tabs-tab",`
 border-top-right-radius: var(--n-tab-border-radius);
 border-bottom-right-radius: var(--n-tab-border-radius);
 `,[n("active",`
 border-left: 1px solid #0000;
 `)]),r("tabs-tab-pad",`
 border-left: 1px solid var(--n-tab-border-color);
 `),r("tabs-pad",`
 border-left: 1px solid var(--n-tab-border-color);
 `)])]),n("bottom",[n("card-type",[r("tabs-scroll-padding","border-top: 1px solid var(--n-tab-border-color);"),m("prefix, suffix",`
 border-top: 1px solid var(--n-tab-border-color);
 `),r("tabs-tab",`
 border-bottom-left-radius: var(--n-tab-border-radius);
 border-bottom-right-radius: var(--n-tab-border-radius);
 `,[n("active",`
 border-top: 1px solid #0000;
 `)]),r("tabs-tab-pad",`
 border-top: 1px solid var(--n-tab-border-color);
 `),r("tabs-pad",`
 border-top: 1px solid var(--n-tab-border-color);
 `)])])])]),jt=Object.assign(Object.assign({},Ce.props),{value:[String,Number],defaultValue:[String,Number],trigger:{type:String,default:"click"},type:{type:String,default:"bar"},closable:Boolean,justifyContent:String,size:{type:String,default:"medium"},placement:{type:String,default:"top"},tabStyle:[String,Object],tabClass:String,addTabStyle:[String,Object],addTabClass:String,barWidth:Number,paneClass:String,paneStyle:[String,Object],paneWrapperClass:String,paneWrapperStyle:[String,Object],addable:[Boolean,Object],tabsPadding:{type:Number,default:0},animated:Boolean,onBeforeLeave:Function,onAdd:Function,"onUpdate:value":[Function,Array],onUpdateValue:[Function,Array],onClose:[Function,Array],labelSize:String,activeName:[String,Number],onActiveNameChange:[Function,Array]}),Dt=se({name:"Tabs",props:jt,slots:Object,setup(e,{slots:s}){var b,w,v,_;const{mergedClsPrefixRef:u,inlineThemeDisabled:y}=Tt(e),C=Ce("Tabs","-tabs",Et,Lt,e,u),h=$(null),c=$(null),k=$(null),L=$(null),g=$(null),S=$(null),x=$(!0),T=$(!0),A=ge(e,["labelSize","size"]),z=ge(e,["activeName","value"]),M=$((w=(b=z.value)!==null&&b!==void 0?b:e.defaultValue)!==null&&w!==void 0?w:s.default?(_=(v=ae(s.default())[0])===null||v===void 0?void 0:v.props)===null||_===void 0?void 0:_.name:null),P=Pt(z,M),d={id:0},R=J(()=>{if(!(!e.justifyContent||e.type==="card"))return{display:"flex",justifyContent:e.justifyContent}});ee(P,()=>{d.id=0,U(),be()});function H(){var t;const{value:a}=P;return a===null?null:(t=h.value)===null||t===void 0?void 0:t.querySelector(`[data-name="${a}"]`)}function Re(t){if(e.type==="card")return;const{value:a}=c;if(!a)return;const o=a.style.opacity==="0";if(t){const i=`${u.value}-tabs-bar--disabled`,{barWidth:f,placement:W}=e;if(t.dataset.disabled==="true"?a.classList.add(i):a.classList.remove(i),["top","bottom"].includes(W)){if(de(["top","maxHeight","height"]),typeof f=="number"&&t.offsetWidth>=f){const B=Math.floor((t.offsetWidth-f)/2)+t.offsetLeft;a.style.left=`${B}px`,a.style.maxWidth=`${f}px`}else a.style.left=`${t.offsetLeft}px`,a.style.maxWidth=`${t.offsetWidth}px`;a.style.width="8192px",o&&(a.style.transition="none"),a.offsetWidth,o&&(a.style.transition="",a.style.opacity="1")}else{if(de(["left","maxWidth","width"]),typeof f=="number"&&t.offsetHeight>=f){const B=Math.floor((t.offsetHeight-f)/2)+t.offsetTop;a.style.top=`${B}px`,a.style.maxHeight=`${f}px`}else a.style.top=`${t.offsetTop}px`,a.style.maxHeight=`${t.offsetHeight}px`;a.style.height="8192px",o&&(a.style.transition="none"),a.offsetHeight,o&&(a.style.transition="",a.style.opacity="1")}}}function ze(){if(e.type==="card")return;const{value:t}=c;t&&(t.style.opacity="0")}function de(t){const{value:a}=c;if(a)for(const o of t)a.style[o]=""}function U(){if(e.type==="card")return;const t=H();t?Re(t):ze()}function be(){var t;const a=(t=g.value)===null||t===void 0?void 0:t.$el;if(!a)return;const o=H();if(!o)return;const{scrollLeft:i,offsetWidth:f}=a,{offsetLeft:W,offsetWidth:B}=o;i>W?a.scrollTo({top:0,left:W,behavior:"smooth"}):W+B>i+f&&a.scrollTo({top:0,left:W+B-f,behavior:"smooth"})}const X=$(null);let Q=0,O=null;function $e(t){const a=X.value;if(a){Q=t.getBoundingClientRect().height;const o=`${Q}px`,i=()=>{a.style.height=o,a.style.maxHeight=o};O?(i(),O(),O=null):O=i}}function Te(t){const a=X.value;if(a){const o=t.getBoundingClientRect().height,i=()=>{document.body.offsetHeight,a.style.maxHeight=`${o}px`,a.style.height=`${Math.max(Q,o)}px`};O?(O(),O=null,i()):O=i}}function Pe(){const t=X.value;if(t){t.style.maxHeight="",t.style.height="";const{paneWrapperStyle:a}=e;if(typeof a=="string")t.style.cssText=a;else if(a){const{maxHeight:o,height:i}=a;o!==void 0&&(t.style.maxHeight=o),i!==void 0&&(t.style.height=i)}}}const ce={value:[]},fe=$("next");function _e(t){const a=P.value;let o="next";for(const i of ce.value){if(i===a)break;if(i===t){o="prev";break}}fe.value=o,We(t)}function We(t){const{onActiveNameChange:a,onUpdateValue:o,"onUpdate:value":i}=e;a&&q(a,t),o&&q(o,t),i&&q(i,t),M.value=t}function Be(t){const{onClose:a}=e;a&&q(a,t)}function pe(){const{value:t}=c;if(!t)return;const a="transition-disabled";t.classList.add(a),U(),t.classList.remove(a)}const F=$(null);function Y({transitionDisabled:t}){const a=h.value;if(!a)return;t&&a.classList.add("transition-disabled");const o=H();o&&F.value&&(F.value.style.width=`${o.offsetWidth}px`,F.value.style.height=`${o.offsetHeight}px`,F.value.style.transform=`translateX(${o.offsetLeft-Bt(getComputedStyle(a).paddingLeft)}px)`,t&&F.value.offsetWidth),t&&a.classList.remove("transition-disabled")}ee([P],()=>{e.type==="segment"&&te(()=>{Y({transitionDisabled:!1})})}),ct(()=>{e.type==="segment"&&Y({transitionDisabled:!0})});let ue=0;function Le(t){var a;if(t.contentRect.width===0&&t.contentRect.height===0||ue===t.contentRect.width)return;ue=t.contentRect.width;const{type:o}=e;if((o==="line"||o==="bar")&&pe(),o!=="segment"){const{placement:i}=e;Z((i==="top"||i==="bottom"?(a=g.value)===null||a===void 0?void 0:a.$el:S.value)||null)}}const Ae=oe(Le,64);ee([()=>e.justifyContent,()=>e.size],()=>{te(()=>{const{type:t}=e;(t==="line"||t==="bar")&&pe()})});const I=$(!1);function Ee(t){var a;const{target:o,contentRect:{width:i,height:f}}=t,W=o.parentElement.parentElement.offsetWidth,B=o.parentElement.parentElement.offsetHeight,{placement:V}=e;if(!I.value)V==="top"||V==="bottom"?W<i&&(I.value=!0):B<f&&(I.value=!0);else{const{value:N}=L;if(!N)return;V==="top"||V==="bottom"?W-i>N.$el.offsetWidth&&(I.value=!1):B-f>N.$el.offsetHeight&&(I.value=!1)}Z(((a=g.value)===null||a===void 0?void 0:a.$el)||null)}const je=oe(Ee,64);function ke(){const{onAdd:t}=e;t&&t(),te(()=>{const a=H(),{value:o}=g;!a||!o||o.scrollTo({left:a.offsetLeft,top:0,behavior:"smooth"})})}function Z(t){if(!t)return;const{placement:a}=e;if(a==="top"||a==="bottom"){const{scrollLeft:o,scrollWidth:i,offsetWidth:f}=t;x.value=o<=0,T.value=o+f>=i}else{const{scrollTop:o,scrollHeight:i,offsetHeight:f}=t;x.value=o<=0,T.value=o+f>=i}}const He=oe(t=>{Z(t.target)},64);ft(le,{triggerRef:E(e,"trigger"),tabStyleRef:E(e,"tabStyle"),tabClassRef:E(e,"tabClass"),addTabStyleRef:E(e,"addTabStyle"),addTabClassRef:E(e,"addTabClass"),paneClassRef:E(e,"paneClass"),paneStyleRef:E(e,"paneStyle"),mergedClsPrefixRef:u,typeRef:E(e,"type"),closableRef:E(e,"closable"),valueRef:P,tabChangeIdRef:d,onBeforeLeaveRef:E(e,"onBeforeLeave"),activateTab:_e,handleClose:Be,handleAdd:ke}),_t(()=>{U(),be()}),pt(()=>{const{value:t}=k;if(!t)return;const{value:a}=u,o=`${a}-tabs-nav-scroll-wrapper--shadow-start`,i=`${a}-tabs-nav-scroll-wrapper--shadow-end`;x.value?t.classList.remove(o):t.classList.add(o),T.value?t.classList.remove(i):t.classList.add(i)});const Oe={syncBarPosition:()=>{U()}},Fe=()=>{Y({transitionDisabled:!0})},ve=J(()=>{const{value:t}=A,{type:a}=e,o={card:"Card",bar:"Bar",line:"Line",segment:"Segment"}[a],i=`${t}${o}`,{self:{barColor:f,closeIconColor:W,closeIconColorHover:B,closeIconColorPressed:V,tabColor:N,tabBorderColor:Ie,paneTextColor:De,tabFontWeight:Ve,tabBorderRadius:Me,tabFontWeightActive:Ne,colorSegment:Ue,fontWeightStrong:Xe,tabColorSegment:Ge,closeSize:Ke,closeIconSize:qe,closeColorHover:Je,closeColorPressed:Qe,closeBorderRadius:Ye,[j("panePadding",t)]:G,[j("tabPadding",i)]:Ze,[j("tabPaddingVertical",i)]:et,[j("tabGap",i)]:tt,[j("tabGap",`${i}Vertical`)]:at,[j("tabTextColor",a)]:rt,[j("tabTextColorActive",a)]:ot,[j("tabTextColorHover",a)]:nt,[j("tabTextColorDisabled",a)]:it,[j("tabFontSize",t)]:st},common:{cubicBezierEaseInOut:lt}}=C.value;return{"--n-bezier":lt,"--n-color-segment":Ue,"--n-bar-color":f,"--n-tab-font-size":st,"--n-tab-text-color":rt,"--n-tab-text-color-active":ot,"--n-tab-text-color-disabled":it,"--n-tab-text-color-hover":nt,"--n-pane-text-color":De,"--n-tab-border-color":Ie,"--n-tab-border-radius":Me,"--n-close-size":Ke,"--n-close-icon-size":qe,"--n-close-color-hover":Je,"--n-close-color-pressed":Qe,"--n-close-border-radius":Ye,"--n-close-icon-color":W,"--n-close-icon-color-hover":B,"--n-close-icon-color-pressed":V,"--n-tab-color":N,"--n-tab-font-weight":Ve,"--n-tab-font-weight-active":Ne,"--n-tab-padding":Ze,"--n-tab-padding-vertical":et,"--n-tab-gap":tt,"--n-tab-gap-vertical":at,"--n-pane-padding-left":K(G,"left"),"--n-pane-padding-right":K(G,"right"),"--n-pane-padding-top":K(G,"top"),"--n-pane-padding-bottom":K(G,"bottom"),"--n-font-weight-strong":Xe,"--n-tab-color-segment":Ge}}),D=y?Wt("tabs",J(()=>`${A.value[0]}${e.type[0]}`),ve,e):void 0;return Object.assign({mergedClsPrefix:u,mergedValue:P,renderedNames:new Set,segmentCapsuleElRef:F,tabsPaneWrapperRef:X,tabsElRef:h,barElRef:c,addTabInstRef:L,xScrollInstRef:g,scrollWrapperElRef:k,addTabFixed:I,tabWrapperStyle:R,handleNavResize:Ae,mergedSize:A,handleScroll:He,handleTabsResize:je,cssVars:y?void 0:ve,themeClass:D==null?void 0:D.themeClass,animationDirection:fe,renderNameListRef:ce,yScrollElRef:S,handleSegmentResize:Fe,onAnimationBeforeLeave:$e,onAnimationEnter:Te,onAnimationAfterEnter:Pe,onRender:D==null?void 0:D.onRender},Oe)},render(){const{mergedClsPrefix:e,type:s,placement:b,addTabFixed:w,addable:v,mergedSize:_,renderNameListRef:u,onRender:y,paneWrapperClass:C,paneWrapperStyle:h,$slots:{default:c,prefix:k,suffix:L}}=this;y==null||y();const g=c?ae(c()).filter(d=>d.type.__TAB_PANE__===!0):[],S=c?ae(c()).filter(d=>d.type.__TAB__===!0):[],x=!S.length,T=s==="card",A=s==="segment",z=!T&&!A&&this.justifyContent;u.value=[];const M=()=>{const d=l("div",{style:this.tabWrapperStyle,class:`${e}-tabs-wrapper`},z?null:l("div",{class:`${e}-tabs-scroll-padding`,style:b==="top"||b==="bottom"?{width:`${this.tabsPadding}px`}:{height:`${this.tabsPadding}px`}}),x?g.map((R,H)=>(u.value.push(R.props.name),ne(l(ie,Object.assign({},R.props,{internalCreatedByPane:!0,internalLeftPadded:H!==0&&(!z||z==="center"||z==="start"||z==="end")}),R.children?{default:R.children.tab}:void 0)))):S.map((R,H)=>(u.value.push(R.props.name),ne(H!==0&&!z?ye(R):R))),!w&&v&&T?me(v,(x?g.length:S.length)!==0):null,z?null:l("div",{class:`${e}-tabs-scroll-padding`,style:{width:`${this.tabsPadding}px`}}));return l("div",{ref:"tabsElRef",class:`${e}-tabs-nav-scroll-content`},T&&v?l(re,{onResize:this.handleTabsResize},{default:()=>d}):d,T?l("div",{class:`${e}-tabs-pad`}):null,T?null:l("div",{ref:"barElRef",class:`${e}-tabs-bar`}))},P=A?"top":b;return l("div",{class:[`${e}-tabs`,this.themeClass,`${e}-tabs--${s}-type`,`${e}-tabs--${_}-size`,z&&`${e}-tabs--flex`,`${e}-tabs--${P}`],style:this.cssVars},l("div",{class:[`${e}-tabs-nav--${s}-type`,`${e}-tabs-nav--${P}`,`${e}-tabs-nav`]},he(k,d=>d&&l("div",{class:`${e}-tabs-nav__prefix`},d)),A?l(re,{onResize:this.handleSegmentResize},{default:()=>l("div",{class:`${e}-tabs-rail`,ref:"tabsElRef"},l("div",{class:`${e}-tabs-capsule`,ref:"segmentCapsuleElRef"},l("div",{class:`${e}-tabs-wrapper`},l("div",{class:`${e}-tabs-tab`}))),x?g.map((d,R)=>(u.value.push(d.props.name),l(ie,Object.assign({},d.props,{internalCreatedByPane:!0,internalLeftPadded:R!==0}),d.children?{default:d.children.tab}:void 0))):S.map((d,R)=>(u.value.push(d.props.name),R===0?d:ye(d))))}):l(re,{onResize:this.handleNavResize},{default:()=>l("div",{class:`${e}-tabs-nav-scroll-wrapper`,ref:"scrollWrapperElRef"},["top","bottom"].includes(P)?l($t,{ref:"xScrollInstRef",onScroll:this.handleScroll},{default:M}):l("div",{class:`${e}-tabs-nav-y-scroll`,onScroll:this.handleScroll,ref:"yScrollElRef"},M()))}),w&&v&&T?me(v,!0):null,he(L,d=>d&&l("div",{class:`${e}-tabs-nav__suffix`},d))),x&&(this.animated&&(P==="top"||P==="bottom")?l("div",{ref:"tabsPaneWrapperRef",style:h,class:[`${e}-tabs-pane-wrapper`,C]},xe(g,this.mergedValue,this.renderedNames,this.onAnimationBeforeLeave,this.onAnimationEnter,this.onAnimationAfterEnter,this.animationDirection)):xe(g,this.mergedValue,this.renderedNames)))}});function xe(e,s,b,w,v,_,u){const y=[];return e.forEach(C=>{const{name:h,displayDirective:c,"display-directive":k}=C.props,L=S=>c===S||k===S,g=s===h;if(C.key!==void 0&&(C.key=h),g||L("show")||L("show:lazy")&&b.has(h)){b.has(h)||b.add(h);const S=!L("if");y.push(S?ut(C,[[vt,g]]):C)}}),u?l(ht,{name:`${u}-transition`,onBeforeLeave:w,onEnter:v,onAfterEnter:_},{default:()=>y}):y}function me(e,s){return l(ie,{ref:"addTabInstRef",key:"__addable",name:"__addable",internalCreatedByPane:!0,internalAddable:!0,internalLeftPadded:s,disabled:typeof e=="object"&&e.disabled})}function ye(e){const s=gt(e);return s.props?s.props.internalLeftPadded=!0:s.props={internalLeftPadded:!0},s}function ne(e){return Array.isArray(e.dynamicProps)?e.dynamicProps.includes("internalLeftPadded")||e.dynamicProps.push("internalLeftPadded"):e.dynamicProps=["internalLeftPadded"],e}export{Dt as N,ie as _,It as a};
