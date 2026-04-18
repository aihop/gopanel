import{k,m as h,z as O,r as E,c as g,o as T,W as C,X as y,Z as b,F as D,a4 as P,q as V,K as z,l as X,al as I,_ as L,$ as j,a5 as M}from"./editor-vue-Ca7D1xBp.js";import{ak as W,d as Z,_ as H}from"./index-D3YKcHFq.js";import{h as _,e as v,f as S,al as $,ce as K,s as U,cG as q,cH as G,u as Y,v as A,cF as J,E as Q,w as ee,x as te,y as ae}from"./naive-ui-data-table-aZaSuRvo.js";const se=_([_("@keyframes badge-wave-spread",{from:{boxShadow:"0 0 0.5px 0px var(--n-ripple-color)",opacity:.6},to:{boxShadow:"0 0 0.5px 4.5px var(--n-ripple-color)",opacity:0}}),v("badge",`
 display: inline-flex;
 position: relative;
 vertical-align: middle;
 font-family: var(--n-font-family);
 `,[S("as-is",[v("badge-sup",{position:"static",transform:"translateX(0)"},[$({transformOrigin:"left bottom",originalTransform:"translateX(0)"})])]),S("dot",[v("badge-sup",`
 height: 8px;
 width: 8px;
 padding: 0;
 min-width: 8px;
 left: 100%;
 bottom: calc(100% - 4px);
 `,[_("::before","border-radius: 4px;")])]),v("badge-sup",`
 background: var(--n-color);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 color: #FFF;
 position: absolute;
 height: 18px;
 line-height: 18px;
 border-radius: 9px;
 padding: 0 6px;
 text-align: center;
 font-size: var(--n-font-size);
 transform: translateX(-50%);
 left: 100%;
 bottom: calc(100% - 9px);
 font-variant-numeric: tabular-nums;
 z-index: 2;
 display: flex;
 align-items: center;
 `,[$({transformOrigin:"left bottom",originalTransform:"translateX(-50%)"}),v("base-wave",{zIndex:1,animationDuration:"2s",animationIterationCount:"infinite",animationDelay:"1s",animationTimingFunction:"var(--n-ripple-bezier)",animationName:"badge-wave-spread"}),_("&::before",`
 opacity: 0;
 transform: scale(1);
 border-radius: 9px;
 content: "";
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)])])]),oe=Object.assign(Object.assign({},A.props),{value:[String,Number],max:Number,dot:Boolean,type:{type:String,default:"default"},show:{type:Boolean,default:!0},showZero:Boolean,processing:Boolean,color:String,offset:Array}),ne=k({name:"Badge",props:oe,setup(e,{slots:o}){const{mergedClsPrefixRef:l,inlineThemeDisabled:m,mergedRtlRef:p}=Y(e),d=A("Badge","-badge",se,W,e,l),u=E(!1),c=()=>{u.value=!0},w=()=>{u.value=!1},s=g(()=>e.show&&(e.dot||e.value!==void 0&&!(!e.showZero&&Number(e.value)<=0)||!J(o.value)));T(()=>{s.value&&(u.value=!0)});const a=Q("Badge",p,l),r=g(()=>{const{type:n,color:i}=e,{common:{cubicBezierEaseInOut:f,cubicBezierEaseOut:R},self:{[ee("color",n)]:x,fontFamily:N,fontSize:F}}=d.value;return{"--n-font-size":F,"--n-font-family":N,"--n-color":i||x,"--n-ripple-color":i||x,"--n-bezier":f,"--n-ripple-bezier":R}}),t=m?te("badge",g(()=>{let n="";const{type:i,color:f}=e;return i&&(n+=i[0]),f&&(n+=ae(f)),n}),r,e):void 0,B=g(()=>{const{offset:n}=e;if(!n)return;const[i,f]=n,R=typeof i=="number"?`${i}px`:i,x=typeof f=="number"?`${f}px`:f;return{transform:`translate(calc(${a!=null&&a.value?"50%":"-50%"} + ${R}), ${x})`}});return{rtlEnabled:a,mergedClsPrefix:l,appeared:u,showBadge:s,handleAfterEnter:c,handleAfterLeave:w,cssVars:m?void 0:r,themeClass:t==null?void 0:t.themeClass,onRender:t==null?void 0:t.onRender,offsetStyle:B}},render(){var e;const{mergedClsPrefix:o,onRender:l,themeClass:m,$slots:p}=this;l==null||l();const d=(e=p.default)===null||e===void 0?void 0:e.call(p);return h("div",{class:[`${o}-badge`,this.rtlEnabled&&`${o}-badge--rtl`,m,{[`${o}-badge--dot`]:this.dot,[`${o}-badge--as-is`]:!d}],style:this.cssVars},d,h(O,{name:"fade-in-scale-up-transition",onAfterEnter:this.handleAfterEnter,onAfterLeave:this.handleAfterLeave},{default:()=>this.showBadge?h("sup",{class:`${o}-badge-sup`,title:K(this.value),style:this.offsetStyle},U(p.value,()=>[this.dot?null:h(q,{clsPrefix:o,appeared:this.appeared,max:this.max,value:this.value})]),this.processing?h(G,{clsPrefix:o}):null):null}))}}),re={class:"mb-6 rounded-[28px]"},le={class:"flex w-full flex-col gap-5 lg:flex-row lg:items-center lg:justify-between"},ie={class:"flex flex-1 flex-wrap gap-3"},de=["value","onChange"],ue={class:"whitespace-nowrap"},ce={class:"flex flex-col gap-2 sm:flex-row sm:items-center"},fe=k({name:"RouterButton",__name:"RouterButton",props:{buttons:{}},emits:["update:active"],setup(e,{emit:o}){const l=e,m=o,p=`RouterButton${new Date().getTime()}`,d=g(()=>l.buttons),u=Z(),c=E("");function w(s){const a=d.value.find(r=>r.label===s);a&&(a.path?u.push({path:a.path}):a.name&&u.push({name:a.name}),c.value=s,m("update:active",s))}return T(()=>{const s=d.value;if(!s.length)return;const a=u.currentRoute.value.path,r=s.find(t=>t.path&&a.startsWith(t.path));c.value=(r==null?void 0:r.label)||s[0].label}),(s,a)=>{const r=ne;return y(),C("div",re,[b("div",le,[b("div",ie,[(y(!0),C(D,null,P(d.value,(t,B)=>(y(),C("label",{key:B,class:z(["cursor-pointer bg-base-accent border-base-accent min-w-[120px] w-full sm:w-auto rounded-[20px] p-1 transition-all duration-200 ease-out hover:-translate-y-[1px] hover:border-blue-300/90 hover:shadow-[0_14px_30px_rgba(59,130,246,0.12)]",[c.value===t.label?"border-blue-500/30 bg-gradient-to-br from-blue-100/95 to-blue-50/92 shadow-[0_16px_34px_rgba(37,99,235,0.14)]":"bg-gradient-to-b from-slate-50/98 to-slate-100/92 dark:bg-base-100"]])},[X(b("input",{"onUpdate:modelValue":a[0]||(a[0]=n=>c.value=n),type:"radio",class:"hidden",name:p,value:t.label,onChange:n=>w(t.label)},null,40,de),[[I,c.value]]),b("div",{class:z(["flex min-h-[54px] sm:min-h-[58px] items-center justify-between gap-3 rounded-2xl px-[18px] text-[15px] font-semibold leading-tight",c.value===t.label?"text-blue-600":"text-slate-600"])},[b("div",ue,M(t.label),1),t.count?(y(),L(r,{key:0,value:t.count,max:999},null,8,["value"])):j("",!0)],2)],2))),128))]),b("div",ce,[V(s.$slots,"route-button",{},void 0,!0)])])])}}}),he=H(fe,[["__scopeId","data-v-660d1abe"]]);export{he as R};
