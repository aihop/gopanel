package request

type SSLCreate struct {
	PrimaryDomain  string `json:"primaryDomain"`
	OtherDomains   string `json:"otherDomains"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	KeyType        string `json:"keyType"`
	Pem            string `json:"pem"`
	PrivateKey     string `json:"privateKey"`
	AcmeAccountID  uint   `json:"acmeAccountId"`
	CloudAccountID uint   `json:"cloudAccountId"`
	DnsAccountID   uint   `json:"dnsAccountId"`
}

type SSLUpdate struct {
	ID          uint   `json:"id"`
	Description string `json:"description"`
	AutoRenew   bool   `json:"autoRenew"`
}

type SSLApply struct {
	WebsiteID uint `json:"websiteId"`
	SSLID     uint `json:"SSLId"`
}

type SSLPushCDN struct {
	SSLID          uint   `json:"sslId"`
	CloudAccountID uint   `json:"cloudAccountId"`
	TargetDomain   string `json:"targetDomain"`
}

type SSLSearch struct {
	Name          string `json:"name"`
	AcmeAccountID string `json:"acmeAccountID"`
}

type AcmeAccountCreate struct {
	Email string `json:"email"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}
