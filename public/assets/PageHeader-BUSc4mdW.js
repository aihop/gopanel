import{h as l,e as z,g as i,a1 as y,cl as w,u as $,v as S,E as R,x as P}from"./naive-ui-data-table-aZaSuRvo.js";import{Z as B}from"./index-DAoVJHEp.js";import{k as E,m as t,c as H}from"./editor-vue-Ca7D1xBp.js";const T=l([z("page-header-header",`
 margin-bottom: 20px;
 `),z("page-header",`
 display: flex;
 align-items: center;
 justify-content: space-between;
 line-height: 1.5;
 font-size: var(--n-font-size);
 `,[i("main",`
 display: flex;
 flex-wrap: nowrap;
 align-items: center;
 `),i("back",`
 display: flex;
 margin-right: 16px;
 font-size: var(--n-back-size);
 cursor: pointer;
 color: var(--n-back-color);
 transition: color .3s var(--n-bezier);
 `,[l("&:hover","color: var(--n-back-color-hover);"),l("&:active","color: var(--n-back-color-pressed);")]),i("avatar",`
 display: flex;
 margin-right: 12px
 `),i("title",`
 margin-right: 16px;
 transition: color .3s var(--n-bezier);
 font-size: var(--n-title-font-size);
 font-weight: var(--n-title-font-weight);
 color: var(--n-title-text-color);
 `),i("subtitle",`
 font-size: 14px;
 transition: color .3s var(--n-bezier);
 color: var(--n-subtitle-text-color);
 `)]),z("page-header-content",`
 font-size: var(--n-font-size);
 `,[l("&:not(:first-child)","margin-top: 20px;")]),z("page-header-footer",`
 font-size: var(--n-font-size);
 `,[l("&:not(:first-child)","margin-top: 20px;")])]),j=Object.assign(Object.assign({},S.props),{title:String,subtitle:String,extra:String,onBack:Function}),V=E({name:"PageHeader",props:j,slots:Object,setup(r){const{mergedClsPrefixRef:o,mergedRtlRef:c,inlineThemeDisabled:s}=$(r),d=S("PageHeader","-page-header",T,B,r,o),e=R("PageHeader",c,o),h=H(()=>{const{self:{titleTextColor:p,subtitleTextColor:g,backColor:f,fontSize:u,titleFontSize:b,backSize:n,titleFontWeight:v,backColorHover:m,backColorPressed:k},common:{cubicBezierEaseInOut:x}}=d.value;return{"--n-title-text-color":p,"--n-title-font-size":b,"--n-title-font-weight":v,"--n-font-size":u,"--n-back-size":n,"--n-subtitle-text-color":g,"--n-back-color":f,"--n-back-color-hover":m,"--n-back-color-pressed":k,"--n-bezier":x}}),a=s?P("page-header",void 0,h,r):void 0;return{rtlEnabled:e,mergedClsPrefix:o,cssVars:s?void 0:h,themeClass:a==null?void 0:a.themeClass,onRender:a==null?void 0:a.onRender}},render(){var r;const{onBack:o,title:c,subtitle:s,extra:d,mergedClsPrefix:e,cssVars:h,$slots:a}=this;(r=this.onRender)===null||r===void 0||r.call(this);const{title:p,subtitle:g,extra:f,default:u,header:b,avatar:n,footer:v,back:m}=a,k=o,x=c||p,_=s||g,C=d||f;return t("div",{style:h,class:[`${e}-page-header-wrapper`,this.themeClass,this.rtlEnabled&&`${e}-page-header-wrapper--rtl`]},b?t("div",{class:`${e}-page-header-header`,key:"breadcrumb"},b()):null,(k||n||x||_||C)&&t("div",{class:`${e}-page-header`,key:"header"},t("div",{class:`${e}-page-header__main`,key:"back"},k?t("div",{class:`${e}-page-header__back`,onClick:o},m?m():t(y,{clsPrefix:e},{default:()=>t(w,null)})):null,n?t("div",{class:`${e}-page-header__avatar`},n()):null,x?t("div",{class:`${e}-page-header__title`,key:"title"},c||p()):null,_?t("div",{class:`${e}-page-header__subtitle`,key:"subtitle"},s||g()):null),C?t("div",{class:`${e}-page-header__extra`},d||f()):null),u?t("div",{class:`${e}-page-header-content`,key:"content"},u()):null,v?t("div",{class:`${e}-page-header-footer`,key:"footer"},v()):null)}});export{V as _};
