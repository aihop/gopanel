package conf

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/aihop/gopanel/cmd"
	"github.com/aihop/gopanel/config"
	configs "github.com/aihop/gopanel/config"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// 初始安装配置
type InitInstallConfig struct {
	BaseDir   string `mapstructure:"base_dir" required:"true"`
	Port      int    `mapstructure:"port" required:"true"`
	User      string `mapstructure:"user" required:"true"`
	Password  string `mapstructure:"password" required:"true"`
	SafeEnter string `mapstructure:"safe_enter" required:"true"`
}

var (
	InitInstall InitInstallConfig
)

// 读取项目根目录下的init.yaml文件，如果存在，则使用init.yaml的配置作为初始配置
func initFile() {
	workDir, _ := os.Getwd()

	// config 目录如果不存在，就创建这个目录
	configDir := path.Join(workDir, "config")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		_ = os.Mkdir(configDir, 0755)
	}

	// 读取项目根目录下的init.yaml文件
	initFilePath := path.Join(workDir, "init.yaml")
	// 如果文件存在，则解析文件映射到结构体
	if _, err := os.Stat(initFilePath); err != nil {
		// 如果文件不存在，则不解析文件
		fmt.Println("not find init.yaml file, skip parse")
		return
	}
	// 解析文件
	v := viper.New()
	v.SetConfigFile(initFilePath)
	v.SetConfigType("yaml")
	_ = v.ReadInConfig()
	_ = v.Unmarshal(&InitInstall)
	fmt.Printf("init install config: %v\n", InitInstall)
	// 删除init.yaml文件
	_ = os.Remove(initFilePath)
}

func Init() {
	initFile()

	p := viper.New()
	p.BindPFlags(pflag.CommandLine)
	// system
	base_dir := "/opt/gopanel" // ...existing code...
	// 读取/写入配置文件：优先使用命令行 --config 指定的路径（若未指定则使用 ./conf.yaml）
	pflag.Parse()

	if f := pflag.Lookup("config"); f != nil {
		if s := f.Value.String(); s != "" {
			cmd.ConfFilePath = s
		}
	}

	p.SetConfigFile(cmd.ConfFilePath)
	p.SetConfigType("yaml")

	// 如果配置已存在则读取，若不存在则延后写入含默认值的新配置文件
	needCreate := false
	if _, err := os.Stat(cmd.ConfFilePath); err == nil {
		if err := p.ReadInConfig(); err != nil {
			fmt.Printf("读取配置文件失败: %v\n", err)
		}
	} else {
		// 确保目录存在
		dir := path.Dir(cmd.ConfFilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("创建配置目录失败: %v\n", err)
		}
		needCreate = true
	}
	if InitInstall.BaseDir != "" {
		base_dir = InitInstall.BaseDir
	}
	p.SetDefault("system.base_dir", base_dir)
	systemPort := 5470
	if InitInstall.Port != 0 {
		systemPort = InitInstall.Port
	}
	p.SetDefault("system.port", ":"+fmt.Sprintf("%d", systemPort))
	p.SetDefault("system.entrance", "default")
	if InitInstall.SafeEnter != "" {
		p.SetDefault("system.entrance", InitInstall.SafeEnter)
	}
	p.SetDefault("system.mode", "dev")
	p.SetDefault("system.version", constant.AppVersion)
	p.SetDefault("system.encrypt_key", common.RandStr(32))

	// log
	p.SetDefault("log.level", "debug")
	p.SetDefault("log.log_name", "default")
	p.SetDefault("log.log_suffix", ".log")
	p.SetDefault("log.max_backup", 10)

	// 在设置完默认值后，若之前标记需要创建，则写入默认配置
	if needCreate {
		if err := p.WriteConfigAs(cmd.ConfFilePath); err != nil {
			fmt.Printf("写入默认配置失败: %v\n", err)
		} else {
			fmt.Printf("已生成默认配置: %s\n", cmd.ConfFilePath)
		}
	} else {
		// 若原来有配置，合并并写回（可选）
		_ = p.ReadInConfig()
		_ = p.WriteConfig()
	}

	// get by env
	p.SetEnvPrefix("gopanel")

	GlobalConfInit(p)
}

func GlobalConfInit(v *viper.Viper) {
	systemConfig := config.System{
		BaseDir:            v.GetString("system.base_dir"),
		Port:               v.GetString("system.port"),
		Mode:               v.GetString("system.mode"),
		Entrance:           v.GetString("system.entrance"),
		Version:            v.GetString("version"),
		EncryptKey:         v.GetString("system.encrypt_key"),
		ApiInterfaceStatus: v.GetString("system.api_interface_status"),
		ApiKey:             v.GetString("system.api_key"),
	}

	logConfig := config.LogConfig{
		Level:     v.GetString("log.level"),
		TimeZone:  v.GetString("log.time_zone"),
		LogName:   v.GetString("log.log_name"),
		LogSuffix: v.GetString("log.log_suffix"),
		MaxBackup: v.GetInt("log.max_backup"),
	}

	global.CONF = configs.ServerConfig{
		Debug:     v.GetBool("debug"),
		System:    systemConfig,
		LogConfig: logConfig,
	}

	global.CONF.System.LicenseVerify = os.Getenv("GOPANEL_LICENSE_VERIFY")
	global.Viper = v

	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			global.CONF.System.BaseDir = filepath.Join(homeDir, ".gopanel")
		}
	}
	global.CONF.System.Cache = path.Join(global.CONF.System.BaseDir, "cache")
	global.CONF.System.Backup = path.Join(global.CONF.System.BaseDir, "backup")
	global.CONF.System.DbPath = path.Join(global.CONF.System.BaseDir, "db")
	global.CONF.System.LogPath = path.Join(global.CONF.System.BaseDir, "log")
	global.CONF.System.TmpDir = path.Join(global.CONF.System.BaseDir, "tmp")
}
