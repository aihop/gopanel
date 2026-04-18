package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/pkg/cloud"
)

// AutoPushCertificate 根据规则自动推送证书到目标云厂商
func AutoPushCertificate(sslID uint) error {
	return AutoPushCertificateWithLogger(sslID, nil)
}

// AutoPushCertificateWithLogger 带有日志输出的自动推送
func AutoPushCertificateWithLogger(sslID uint, logger *SSLLogger) error {
	sslRepo := repo.NewSSL()
	ruleRepo := repo.NewSSLPushRule()
	cloudAccountRepo := repo.NewCloudAccount()

	// 1. 获取证书信息
	ssl, err := sslRepo.GetFirst(sslRepo.WithID(sslID))
	if err != nil {
		if logger != nil {
			logger.Error("读取待部署证书信息失败: %v", err)
		}
		return fmt.Errorf("certificate not found: %v", err)
	}

	// 2. 获取该证书绑定的所有推送规则
	rules, err := ruleRepo.GetBy(ruleRepo.WithSSLID(sslID))
	if err != nil || len(rules) == 0 {
		if logger != nil {
			logger.Info("未发现针对此证书的自动部署规则，跳过推送。")
		}
		return nil // 无规则，不执行推送
	}

	if logger != nil {
		logger.Info("发现 %d 条自动部署规则，开始执行推送到云厂商...", len(rules))
	}

	var errs []string

	// 3. 遍历规则并推送
	for _, rule := range rules {
		if logger != nil {
			logger.Info("-> [规则 #%d] 正在读取目标云账号信息...", rule.ID)
		}

		cloudAccount, err := cloudAccountRepo.GetByID(rule.CloudAccountID)
		if err != nil {
			errs = append(errs, fmt.Sprintf("rule %d: dns account not found", rule.ID))
			rule.Status = "error"
			rule.Message = "云账号不存在"
			if logger != nil {
				logger.Error("   目标云账号未找到或已被删除！")
			}
			ruleRepo.Save(context.Background(), &rule)
			continue
		}

		targetDomain := rule.TargetDomain
		if targetDomain == "" {
			targetDomain = ssl.PrimaryDomain
		}

		if logger != nil {
			logger.Info("   目标云服务商: %s, 部署域名: %s", cloudAccount.Type, targetDomain)
			logger.Info("   正在调用 %s CDN 接口上传新证书并关联域名...", cloudAccount.Type)
		}

		var authData map[string]interface{}
		_ = json.Unmarshal([]byte(cloudAccount.Authorization), &authData)

		provider, err := cloud.NewProvider(cloudAccount.Type, authData)
		if err != nil {
			pushErr := fmt.Errorf("provider initialization failed: %v", err)
			errs = append(errs, fmt.Sprintf("rule %d: %v", rule.ID, pushErr))
			rule.Status = "error"
			rule.Message = pushErr.Error()
			if logger != nil {
				logger.Error("   ❌ 初始化云服务提供者失败: %v", pushErr)
			}
			ruleRepo.Save(context.Background(), &rule)
			continue
		}

		certName := fmt.Sprintf("gopanel-%s-%d", strings.ReplaceAll(targetDomain, ".", "-"), ssl.ID)
		pushErr := provider.SetCdnDomainSSLCertificate(targetDomain, certName, ssl.Pem, ssl.PrivateKey)

		if pushErr != nil {
			errs = append(errs, fmt.Sprintf("rule %d: %v", rule.ID, pushErr))
			rule.Status = "error"
			rule.Message = pushErr.Error()
			if logger != nil {
				logger.Error("   ❌ 部署失败: %v", pushErr)
			}
		} else {
			rule.Status = "success"
			rule.Message = "推送成功"
			if logger != nil {
				logger.Info("   ✅ 成功将新证书部署至 %s", targetDomain)
			}
		}
		ruleRepo.Save(context.Background(), &rule)
	}

	if len(errs) > 0 {
		return fmt.Errorf("auto push completed with errors: %s", strings.Join(errs, "; "))
	}
	return nil
}
