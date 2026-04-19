package env

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/viper" // 统一使用 Viper
)

func Write(envMap map[string]string, filename string) error {
	content, err := Marshal(envMap)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content + "\n")
	if err != nil {
		return err
	}
	return file.Sync()
}

func Marshal(envMap map[string]string) (string, error) {
	lines := make([]string, 0, len(envMap))
	for k, v := range envMap {
		// 保持原有的逻辑：数字直接写，字符串带引号
		if d, err := strconv.Atoi(v); err == nil {
			lines = append(lines, fmt.Sprintf(`%s=%d`, k, d))
		} else {
			lines = append(lines, fmt.Sprintf(`%s="%s"`, k, v))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

func GetEnvValueByKey(envPath, key string) (string, error) {
	v := viper.New()
	v.SetConfigFile(envPath)
	v.SetConfigType("env") // 明确指定按 env 格式解析

	if err := v.ReadInConfig(); err != nil {
		return "", fmt.Errorf("failed to read env file: %w", err)
	}

	// Viper 默认不区分大小写，如果你的 Key 必须严格区分，
	// 注意 Viper 的 GetString 会返回空字符串（如果不存在）
	if !v.IsSet(key) {
		return "", fmt.Errorf("key %s not found in %s", key, envPath)
	}

	return v.GetString(key), nil
}
