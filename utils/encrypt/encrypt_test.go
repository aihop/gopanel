package encrypt

import (
	"strings"
	"testing"

	"github.com/aihop/gopanel/global"
)

// 回归背景：节点令牌用旧 key 加密后换了 key，解密得到随机字节，
// unPadding 无校验切片直接 panic，把整个面板进程打崩（cron goroutine 无 recover）。
// 这里锁死契约：任何坏输入都只能返回 error，不允许 panic。

func setKey(t *testing.T, key string) {
	t.Helper()
	old := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = key
	t.Cleanup(func() { global.CONF.System.EncryptKey = old })
}

func TestStringEncryptDecryptRoundtrip(t *testing.T) {
	setKey(t, "0123456789abcdef0123456789abcdef")
	plain := "node-token-40-chars-1234567890abcdefghij"
	cipherText, err := StringEncrypt(plain)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	got, err := StringDecrypt(cipherText)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if got != plain {
		t.Fatalf("roundtrip 不一致: got=%q want=%q", got, plain)
	}
}

func TestStringDecryptWrongKeyReturnsError(t *testing.T) {
	setKey(t, "0123456789abcdef0123456789abcdef")
	cipherText, err := StringEncrypt("secret-value")
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 换 key 解密：结果是随机字节。多数情况下填充校验会报错；
	// 极小概率末字节恰好构成合法填充,此时解出乱码但同样不允许 panic。
	setKey(t, "ffffffffffffffffffffffffffffffff")
	if out, err := StringDecrypt(cipherText); err == nil {
		t.Logf("错误 key 未报错（随机字节恰好像合法填充），解出乱码: %q", out)
	}
}

func TestStringDecryptCorruptedInputReturnsError(t *testing.T) {
	setKey(t, "0123456789abcdef0123456789abcdef")
	cases := map[string]string{
		"非 base64":  "!!!not-base64!!!",
		"过短密文":      "QUJD", // "ABC"，短于一个块
		"非整块长度":     "QUJDREVGR0hJSktMTU5PUFFS", // 18 字节
		"全零块伪造填充越界": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}
	for name, input := range cases {
		if _, err := StringDecrypt(input); err == nil && !strings.Contains(name, "伪造") {
			t.Errorf("%s: 期望返回错误,实际 err=nil", name)
		}
		// 不 panic 本身就是断言:任何一例 panic 都会让整个测试进程失败
	}
}
