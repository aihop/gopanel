package docker

import (
	"bytes"
	"context"

	"path"
	"regexp"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/spf13/viper"
)

// 辅助函数：使用 Viper 解析 env 字节流为 map
func parseEnvMap(env []byte) (map[string]string, error) {
	if len(env) == 0 {
		return make(map[string]string), nil
	}

	v := viper.New()
	v.SetConfigType("env")
	if err := v.ReadConfig(bytes.NewReader(env)); err != nil {
		return nil, err
	}

	// 将 viper 的配置转换为 compose 需要的 map[string]string
	envMap := make(map[string]string)
	for k, v := range v.AllSettings() {
		envMap[strings.ToUpper(k)] = v.(string)
	}
	return envMap, nil
}

func GetComposeProject(projectName, workDir string, yml []byte, env []byte, skipNormalization bool) (*types.Project, error) {
	var configFiles []types.ConfigFile
	configFiles = append(configFiles, types.ConfigFile{
		Filename: "docker-compose.yml",
		Content:  yml},
	)

	// 使用 viper 替代 godotenv
	envMap, err := parseEnvMap(env)
	if err != nil {
		return nil, err
	}

	details := types.ConfigDetails{
		WorkingDir:  workDir,
		ConfigFiles: configFiles,
		Environment: envMap,
	}

	projectName = strings.ToLower(projectName)
	reg, _ := regexp.Compile(`[^a-z0-9_-]+`)
	projectName = reg.ReplaceAllString(projectName, "")

	project, err := loader.LoadWithContext(context.Background(), details, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
		options.ResolvePaths = true
		options.SkipNormalization = skipNormalization
	})
	if err != nil {
		return nil, err
	}

	project.ComposeFiles = []string{path.Join(workDir, "docker-compose.yml")}
	return project, nil
}

type ComposeProject struct {
	Version  string
	Services map[string]Service `yaml:"services"`
}

type Service struct {
	Image string `yaml:"image"`
}

func GetDockerComposeImages(projectName string, env, yml []byte) ([]string, error) {
	var (
		configFiles []types.ConfigFile
		images      []string
	)
	configFiles = append(configFiles, types.ConfigFile{
		Filename: "docker-compose.yml",
		Content:  yml},
	)
	envMap, err := parseEnvMap(env)
	if err != nil {
		return nil, err
	}
	details := types.ConfigDetails{
		ConfigFiles: configFiles,
		Environment: envMap,
	}

	project, err := loader.LoadWithContext(context.Background(), details, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
		options.ResolvePaths = true
	})
	if err != nil {
		return nil, err
	}
	for _, service := range project.AllServices() {
		images = append(images, service.Image)
	}
	return images, nil
}
