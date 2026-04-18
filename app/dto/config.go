/*
 * @Author: coller
 * @Date: 2024-04-01 16:12:23
 * @LastEditors: coller
 * @LastEditTime: 2024-04-17 23:21:11
 * @Desc: 配置
 */
package dto

type ConfigID struct {
	ID     string   `json:"id" validate:"required"` // ID
	Params []string `json:"params"`
}

type ConfigSSL struct {
	Email  string `json:"email" validate:"required"`
	Domain string `json:"domain" validate:"required"`
}

type LangType struct {
	CountryCode string `json:"countryCode"`
	Name        string `json:"name"`
	Lang        string `json:"lang"`
	BrowserLang string `json:"browserLang"`
}

type CurrencyType struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}
