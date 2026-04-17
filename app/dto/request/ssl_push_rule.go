package request

type SSLPushRuleCreate struct {
	SSLID          uint   `json:"sslId" validate:"required"`
	CloudAccountID uint   `json:"cloudAccountId" validate:"required"`
	TargetDomain   string `json:"targetDomain"`
}

type SSLPushRuleUpdate struct {
	ID             uint   `json:"id" validate:"required"`
	CloudAccountID uint   `json:"cloudAccountId"`
	TargetDomain   string `json:"targetDomain"`
}
