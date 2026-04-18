package dto

type LocalAppItem struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type LocalAppFormField struct {
	Default  interface{} `json:"default" yaml:"default"`
	EnvKey   string      `json:"envKey" yaml:"envKey"`
	LabelZh  string      `json:"labelZh" yaml:"labelZh"`
	LabelEn  string      `json:"labelEn" yaml:"labelEn"`
	Rule     string      `json:"rule" yaml:"rule"`
	Type     string      `json:"type" yaml:"type"`
	Required bool        `json:"required" yaml:"required"`
	Random   bool        `json:"random" yaml:"random"`
	Values   interface{} `json:"values" yaml:"values"`
	Multiple bool        `json:"multiple" yaml:"multiple"`
}

type LocalAppDetail struct {
	Key        string              `json:"key"`
	Name       string              `json:"name"`
	FormFields []LocalAppFormField `json:"formFields"`
	Compose    string              `json:"compose"`
	Env        map[string]string   `json:"env"`
}
