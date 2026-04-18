package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/cryptx"
	"github.com/glebarez/sqlite"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var (
	ConfFilePath  string // 配置文件路径
	versionFlag   bool   // 版本标志
	helpFlag      bool   // 帮助标志
	showConfig    bool   // 显示安全配置标志
	resetPassword bool   // 重置密码标志
)

func Init() {
	log.Println("GoPanel is starting...")
	pflag.StringVarP(&ConfFilePath, "config", "c", "./conf.yaml", "config file path.")
	pflag.BoolVarP(&versionFlag, "version", "v", false, "show version info")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "show help information")
	pflag.BoolVarP(&showConfig, "show-config", "s", false, "show security configuration")
	pflag.BoolVarP(&resetPassword, "reset-password", "r", false, "reset super user password")
	pflag.Parse()

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GoPanel - 容器化应用管理平台\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		pflag.PrintDefaults()
	}

	if helpFlag {
		pflag.Usage()
		os.Exit(0)
	}

	if versionFlag {
		fmt.Printf("Version: %s\nBuild Date: %s\n", constant.AppVersion, constant.BuildTime)
		os.Exit(0)
	}

	if showConfig {
		showSecurityConfig()
		os.Exit(0)
	}

	if resetPassword {
		resetSuperUserPassword()
		os.Exit(0)
	}
}

func showSecurityConfig() {
	// 读取配置文件
	viper.SetConfigFile(ConfFilePath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}
	// 获取安全入口和运行端口
	securityEntry := viper.GetString("system.entrance")
	serverPort := viper.GetString("http.listen")

	// 显示配置信息
	fmt.Printf("Security Entry: %s\n", securityEntry)
	fmt.Printf("Server Port: %s\n", serverPort)
}

func resetSuperUserPassword() {
	viper.SetConfigFile(ConfFilePath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	// 获取数据库配置
	dbDatabase := viper.GetString("db.database")

	// 连接数据库
	dsn := dbDatabase
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 查询用户信息（显式指定表名为user）
	var user model.User
	if err := db.Table("user").Where("status = 20").Where("role = ?", "SUPER").First(&user).Error; err != nil {
		log.Fatalf("Failed to query user: %v", err)
	}

	newPassword := ""

	fmt.Printf("Reset %s Password\n", user.Email)

	fmt.Println("Please input new password: ")
	fmt.Scan(&newPassword)
	if newPassword == "" {
		fmt.Println("password can not be empty")
		return
	}

	user.Password = cryptx.EncodePassword(newPassword)
	if err = db.Table("user").Where("id = ?", user.ID).Update("password", user.Password).Error; err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	fmt.Println("Reset Password Success")

	fmt.Println("-----------------------------------------")
	fmt.Println("Please use the new password to login.")
	fmt.Printf("Email: %s\n", user.Email)
	fmt.Printf("Password: %s\n", newPassword)
	fmt.Println("-----------------------------------------")
}
