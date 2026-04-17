package spaceship

import "errors"

func ListCdnDomains(apiKey, apiSecret string) ([]string, error) {
	// 目前通过 spaceship api 还没有查询到直接列出所有绑定域名的 API 文档，可以先返回空或者支持的域名列表。
	// 在没有明确官方 API 的情况下，先返回 not supported 或通过其它方式列出所有 record。
	return nil, errors.New("spaceship api not support list cdn domains")
}

func SetCdnDomainSSLCertificate(apiKey, apiSecret, domainName, certName, sslPub, sslPri string) error {
	// spaceship 主要作为域名解析商，通常不作为 CDN，因此不支持给 CDN 域名上传 SSL 证书
	return errors.New("spaceship not support set cdn ssl certificate")
}
