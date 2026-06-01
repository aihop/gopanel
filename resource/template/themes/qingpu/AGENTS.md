# 轻铺（QingPu）H5 模板主题

## 一句话定位

`qingpu` 是 GoPanel 的轻量 H5 店铺模板主题，面向移动端展示型店铺页面，对标微店风格。核心目标：**SEO 友好 + AI 爬虫可抓取结构化数据**。

## 给 AI 的最高优先级指令

- 模板只做中文，不做多语言
- 所有页面必须是服务端渲染（SSR），不能依赖客户端 JS 渲染关键内容
- 页面输出语义化 HTML5 + JSON-LD 结构化数据
- H5 优先（320px-750px 断点），桌面端只做基础兼容（>750px 居中限宽）
- 优先沿 shoply `template/common` 的机制走，不平地起新体系

---

## 模板引擎机制（继承自 Shoply）

### 渲染语言

Go 标准库 `html/template`，模板文件中使用 `{{ }}` 语法。

### 模板入口

```
resource/template/themes/qingpu/
├── theme.json              ← 主题元信息
├── layout/                 ← 布局骨架
│   ├── head.html           ← <head> 输出 (SEO meta/OG/Twitter)
│   └── foot.html           ← 页脚脚本
├── page/                   ← 页面模板
│   ├── shop.html           ← 店铺首页
│   ├── product.html        ← 商品详情
│   ├── list.html           ← 商品列表
│   └── about.html          ← 关于我们
├── block/                  ← 可复用区块
│   ├── product_card.html   ← 商品卡片
│   ├── banner.html         ← 轮播横幅
│   ├── header.html         ← 店铺头部
│   └── footer.html         ← 店铺底部
├── static/                 ← 静态资源
│   ├── css/main.css
│   ├── js/main.js
│   └── images/
└── AGENTS.md
```

### 模板语法约定

参考 shoply 的 `FT::component` / `FT::diySlot` 机制：

```
<!--FT::diySlot:pageHead-->          ← 可插拔插槽
  <!--FT::component:ft_header-->     ← 组件引用
<!--/FT::diySlot:pageHead-->
```

**页面结构（每个 page/*.html 遵循）：**

```
<!--FT::pageHead-->
  <!--FT::component:header-->
<!--/FT::pageHead-->

<!--FT::pageBody-->
  <!--FT::component:banner-->
  <!--FT::component:product_card-->
<!--/FT::pageBody-->

<!--FT::pageFoot-->
  <!--FT::component:footer-->
<!--/FT::pageFoot-->
```

### 模板变量契约

| 变量 | 来源 | 说明 |
|------|------|------|
| `{{.Shop.Name}}` | 店铺数据 | 店铺名称 |
| `{{.Shop.Logo}}` | 店铺数据 | 店铺 Logo URL |
| `{{.Shop.Description}}` | 店铺数据 | 店铺描述 |
| `{{.Shop.Contact}}` | 店铺数据 | 联系方式 |
| `{{.Products}}` | 商品数据 | `[]Product` 商品列表 |
| `{{.Product}}` | 商品数据 | 当前商品详情 |
| `{{.Banners}}` | 店铺数据 | `[]Banner` 轮播图 |
| `{{.Categories}}` | 分类数据 | `[]Category` 商品分类 |
| `{{.Meta.Title}}` | SEO | 页面标题 |
| `{{.Meta.Description}}` | SEO | 页面描述 |
| `{{.Meta.Keywords}}` | SEO | 页面关键词 |
| `{{.Meta.Thumbnail}}` | SEO | OG 缩略图 |
| `{{.Meta.URL}}` | SEO | 当前页面 URL |
| `{{.Config}}` | 系统 | 站点配置 |

**Product 结构：**
```
Product {
  ID: int
  Name: string
  Price: float64
  OriginalPrice: float64
  Images: []string
  Description: string
  Stock: int
  Sales: int
  CategoryName: string
  CreatedAt: string
}
```

---

## SEO 硬性要求

### 每个页面必须输出

```
<title>{{.Meta.Title}}</title>
<meta name="description" content="{{.Meta.Description}}">
<meta name="keywords" content="{{.Meta.Keywords}}">
<meta property="og:title" content="{{.Meta.Title}}">
<meta property="og:description" content="{{.Meta.Description}}">
<meta property="og:image" content="{{.Meta.Thumbnail}}">
<meta property="og:type" content="website">
<meta name="twitter:card" content="summary_large_image">
<link rel="canonical" href="{{.Meta.URL}}">
```

### JSON-LD 结构化数据

**店铺首页输出 `Organization` / `Store` Schema：**
```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Store",
  "name": "{{.Shop.Name}}",
  "description": "{{.Shop.Description}}",
  "image": "{{.Shop.Logo}}",
  "url": "{{.Meta.URL}}"
}
</script>
```

**商品详情页输出 `Product` Schema：**
```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Product",
  "name": "{{.Product.Name}}",
  "image": "{{.Product.Images}}",
  "description": "{{.Product.Description}}",
  "offers": {
    "@type": "Offer",
    "price": "{{.Product.Price}}",
    "priceCurrency": "CNY",
    "availability": "{{if .Product.Stock}}InStock{{else}}OutOfStock{{end}}"
  }
}
</script>
```

### 语义化 HTML

- 店铺名用 `<h1>`，区块标题用 `<h2>`，商品名用 `<h3>`
- 导航用 `<nav>`，主体内容用 `<main>`，文章用 `<article>`
- 图片必须带 `alt` 属性
- 合理的 `<meta viewport>` 配置

---

## H5 视觉规范

```
设计风格：简约白底，绿色主色调 (#07c160)
圆角风格：12px-16px 圆角
字体：系统默认字体栈
卡片阴影：轻微 (0 2px 8px rgba(0,0,0,0.06))
间距：16px 基准间距

移动端 (<750px)：全宽布局，边距 16px
桌面端 (>750px)：居中容器，最大宽度 640px
```

### 页面结构（参考微店风格）

```
┌──────────────────┐
│  店铺头部          │  ← Logo + 店铺名 + 关注
│  ┌──────────────┐ │
│  │  轮播 Banner  │ │  ← 3-5 张，自动轮播
│  └──────────────┘ │
│  分类导航          │  ← 横向滚动，圆角标签
│  ┌────┐ ┌────┐   │
│  │商品│ │商品│   │  ← 2 列卡片网格
│  │    │ │    │   │
│  └────┘ └────┘   │
│  加载更多 / 分页   │
│  底部信息          │  ← 联系方式 + 版权
└──────────────────┘
```

---

## 组件清单

| 组件 ID | 文件 | 说明 |
|---------|------|------|
| `header` | `block/header.html` | 店铺头部（Logo + 店名 + 导航） |
| `banner` | `block/banner.html` | 轮播横幅 |
| `product_card` | `block/product_card.html` | 商品卡片（2 列网格） |
| `product_list` | `block/product_list.html` | 商品列表行 |
| `category_nav` | `block/category_nav.html` | 分类导航标签 |
| `footer` | `block/footer.html` | 底部信息 |
| `pagination` | `block/pagination.html` | 分页器 |
| `product_schema` | `block/product_schema.html` | JSON-LD 结构化数据 |

---

## 页面清单

| 路由 | 模板 | SEO 重点 |
|------|------|------|
| `/` | `page/shop.html` | Store Schema |
| `/list` | `page/list.html` | 分页、筛选 |
| `/product/:id` | `page/product.html` | Product Schema |
| `/about` | `page/about.html` | ContactPoint |

---

## 开发约束

### 不要做的事

- 不要把多语言做进去 —— 只做中文
- 不要依赖客户端 JS 渲染关键内容 —— SSR 必须是完整的
- 不要用 Vue/React SPA 方式 —— 用 Go html/template
- 不要引入重量级 CSS 框架 —— 用 TailwindCSS 或轻量自定义 CSS
- 不要忽略 SEO 元标签 —— 每个页面必须完整输出
- 不要在模板里硬编码文案 —— 后续可以统一管理

### 验证清单

每次改完模板后检查：
1. `curl` 返回的 HTML 是否包含完整的 `<title>` 和 `<meta description>`
2. 移动端 viewport 下布局是否正常
3. JSON-LD Schema 是否合法（Google Rich Results Test）
4. 所有图片是否带有 `alt` 属性
5. 不存在客户端空状态（如 "加载中..."）出现在 SSR 输出中

### 参考

- Shoply `template/common/` 的模板机制
- Go 标准库 `html/template` 文档
- schema.org/Product
- schema.org/Store
