package cloud

import (
	"fmt"

	"github.com/go-acme/lego/v4/challenge"
)

func getFirstString(authData map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := authData[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

// Provider 定义了云服务商的统一接口
type Provider interface {
	// GetDNSProvider 获取用于 ACME 的 DNS-01 验证提供者
	GetDNSProvider() (challenge.Provider, error)

	// ListCdnDomains 获取当前云账号下的所有 CDN 域名
	ListCdnDomains() ([]string, error)

	// SetCdnDomainSSLCertificate 为指定的 CDN 域名设置 SSL 证书
	SetCdnDomainSSLCertificate(domainName, certName, sslPub, sslPri string) error
}

// Factory 根据类型和认证数据创建对应的 Provider
func NewProvider(providerType string, authData map[string]interface{}) (Provider, error) {
	switch providerType {
	case "aliyun":
		ak := getFirstString(authData, "accessKey")
		sk := getFirstString(authData, "secretKey")
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("aliyun accessKey or secretKey is empty")
		}
		return NewAliyunProvider(ak, sk), nil
	case "tencentcloud":
		secretID := getFirstString(authData, "secretId", "SecretId", "accessKey")
		secretKey := getFirstString(authData, "secretKey", "SecretKey")
		if secretID == "" || secretKey == "" {
			return nil, fmt.Errorf("tencentcloud secretId or secretKey is empty")
		}
		return NewTencentCloudProvider(secretID, secretKey), nil
	case "cloudflare":
		token := getFirstString(authData, "token")
		if token == "" {
			return nil, fmt.Errorf("cloudflare token is empty")
		}
		return NewCloudflareProvider(token), nil
	case "huaweicloud":
		ak := getFirstString(authData, "accessKey")
		sk := getFirstString(authData, "secretKey")
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("huaweicloud accessKey or secretKey is empty")
		}
		return NewHuaweiCloudProvider(ak, sk), nil
	case "aws":
		ak := getFirstString(authData, "accessKey")
		sk := getFirstString(authData, "secretKey")
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("aws accessKey or secretKey is empty")
		}
		return NewAWSProvider(ak, sk), nil
	case "spaceship":
		ak := getFirstString(authData, "accessKey", "apiKey")
		sk := getFirstString(authData, "secretKey", "apiSecret")
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("spaceship apiKey or apiSecret is empty")
		}
		return NewSpaceshipProvider(ak, sk), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}
