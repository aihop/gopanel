package service

import (
	"encoding/json"
	"fmt"

	"github.com/aihop/gopanel/pkg/cloud"
)

// FetchCdnDomains 根据云账号 ID 调用对应厂商的 API 拉取 CDN 域名列表
func (s *CloudAccountService) CdnDomains(cloudAccountId uint) ([]string, error) {
	cloudAccount, err := s.repo.GetByID(cloudAccountId)
	if err != nil {
		return nil, fmt.Errorf("云账号不存在")
	}

	var authData map[string]interface{}
	if err := json.Unmarshal([]byte(cloudAccount.Authorization), &authData); err != nil {
		return nil, fmt.Errorf("云账号授权信息解析失败")
	}

	provider, err := cloud.NewProvider(cloudAccount.Type, authData)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloud provider: %v", err)
	}

	return provider.ListCdnDomains()
}
