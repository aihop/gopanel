# API 签名 v2

写请求必须使用 v2 HMAC-SHA256 签名。旧 MD5 签名仅兼容 `GET`、`HEAD` 请求。

请求需携带 `apiKey`、`timestamp`、`nonce`、`signatureVersion=v2`，可放请求头或查询参数。`nonce` 必须在签名有效期内唯一。

签名原文按以下字段用换行符连接：

```text
timestamp
nonce
大写 HTTP 方法
URL 路径
规范化查询串
请求体 SHA256 十六进制值
```

规范化查询串使用 URL 编码并按 key 排序，排除 `apiKey`、`timestamp`、`nonce`、`signatureVersion` 四个鉴权字段。使用面板 API Token 作为 HMAC-SHA256 密钥，输出小写十六进制字符串作为 `apiKey`。

签名覆盖 HTTP 方法、路径、业务查询参数和原始请求体；任何字段变化都必须重新签名。
