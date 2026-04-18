package gormx

import "gorm.io/gorm"

type Contextx struct {
	Limit      int           `json:"limit"`      // 条数
	Page       int           `json:"page"`       // 页数
	Select     string        `json:"select"`     // 字段
	Order      string        `json:"order"`      // 排序 如 id DESC降序 ASC升序
	Conditions [][]*WhereOne `json:"conditions"` // 复杂查询条件 conditions 数组对象嵌套对象
	Wheres     []*WhereOne   `json:"wheres"`     // 一般查询条件 wheres 数组对象
	Preloads   []*Preloadx   `json:"preloads"`   // 预加载 数组嵌套对象数组
	Joins      []*Preloadx   `json:"joins"`      // 关联查询 数组嵌套对象数据
	Comment    string        `json:"comment"`
	Table      string
	Scope      []func(*gorm.DB) *gorm.DB
	Extends    map[string]interface{} `json:"extends"`    // 用于接收其他字段
	SiteURL    string                 `json:"siteURL"`    // 域名，不包含/
	ExcludeIds []uint                 `json:"excludeIds"` // 排除的ID
}

type Wherex struct {
	Conditions [][]*WhereOne `json:"conditions"` // 复杂查询条件 conditions 数组对象嵌套对象
	Wheres     []*WhereOne   `json:"wheres"`     // 一般查询条件 wheres 数组对象
	Joins      []*Preloadx   `json:"joins"`      // 关联查询 数组嵌套对象数据
	Table      string
	Select     string `json:"select"` // 字段
}

type Preloadx struct {
	Table  string      `json:"table"`
	Wheres []*WhereOne `json:"wheres"` // 查询条件wheres
	Order  string      `json:"order"`  // 排序 如 id DESC降序 ASC升序
}

func NewContextx(limit int, order string) *Contextx {
	return &Contextx{Limit: limit, Order: order}
}

func NewContextWithComment(limit int, order, comment string) *Contextx {
	return &Contextx{Limit: limit, Order: order, Comment: comment}
}

func NewContextByComment(comment string) *Contextx {
	return &Contextx{Comment: comment}
}

type WhereOne struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Val   string `json:"val"`
}

type WhereRule = string

const (
	WhereRuleEq      = "eq"      // 等于
	WhereRuleNEq     = "nEq"     // 不等于
	WhereRuleEqTrue  = "eqTrue"  // 等于真
	WhereRuleEqFalse = "eqFalse" // 等于假
	WhereRuleGt      = "gt"      // 大于
	WhereRuleGtE     = "gtEq"    // 大于等于
	WhereRuleLt      = "lt"      // 小于
	WhereRuleLtE     = "ltEq"    // 小于等于
	WhereRuleNull    = "null"    // 为空
	WhereRuleNNull   = "nNull"   // 不为空
	WhereRuleIn      = "in"      // 在
	WhereRuleNIn     = "nIn"     // 不在
	WhereRuleInInt   = "inInt"   // 在int
	WhereRuleNInInt  = "nInInt"  // 不在int
	WhereRuleLikes   = "likes"   // 包含多个
	WhereRuleNLikes  = "nLikes"  // 不包含多个
	WhereRuleLike    = "like"    // 包含
	WhereRuleNLike   = "nLike"   // 不包含
	WhereRuleLikeBf  = "likeBf"  // 包含前
	WhereRuleLikeAf  = "likeAf"  // 包含后
	WhereRuleBtw     = "btw"     // 区间
	WhereRuleNBtw    = "nBtw"    // 不在区间
	WhereRuleJArr    = "jArr"    // JsonArr
	WhereRuleJObj    = "jObj"    // JsonArr

	WhereRuleJLikes = "jLikes" // Json包含多个
)
