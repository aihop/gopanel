package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/cryptx"
	"github.com/glebarez/sqlite"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/term"
	"gorm.io/gorm"
)

var (
	ConfFilePath  string // 配置文件路径
	helpFlag      bool
	versionFlag   bool // 版本标志
	showConfig    bool // 显示安全配置标志
	resetPassword bool // 重置密码标志
)

func Init() {
	log.Println("GoPanel is starting...")
	pflag.StringVarP(&ConfFilePath, "config", "c", "./conf.yaml", "config file path.")
	pflag.BoolVarP(&helpFlag, "help", "h", false, "show help")
	pflag.BoolVarP(&versionFlag, "version", "v", false, "show version info")
	pflag.BoolVarP(&showConfig, "show-config", "s", false, "show security configuration")
	pflag.BoolVarP(&resetPassword, "reset-password", "r", false, "reset a panel account password, optionally: --reset-password <email>")
	pflag.Parse()

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "GoPanel - 容器化应用管理平台\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [email]\n\nOptions:\n", os.Args[0])
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n重置密码:\n")
		fmt.Fprintf(os.Stderr, "  %s --reset-password              # 重置超级管理员\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --reset-password user@a.com   # 重置指定账号\n", os.Args[0])
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
	// 获取安全入口和运行端口（运行端口的唯一来源是 system.port，http.listen 只是历史遗留写法）
	securityEntry := viper.GetString("system.entrance")
	serverPort := viper.GetString("system.port")
	if strings.TrimSpace(serverPort) == "" {
		serverPort = viper.GetString("http.listen")
	}

	// 显示配置信息
	fmt.Printf("Security Entry: %s\n", securityEntry)
	fmt.Printf("Server Port: %s\n", serverPort)
}

// 重置面板账号密码。
// 用法：gopanel --reset-password [邮箱]
// 不带邮箱时重置唯一的超级管理员；库里有多个 SUPER 时会要求显式指定邮箱。
func resetSuperUserPassword() {
	viper.SetConfigFile(ConfFilePath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败: %s", err)
	}

	// 数据库位置必须和面板运行时一致：<base_dir>/db/gopanel.db（见 init/db.Init）。
	// 早期这里读的是 conf.yaml 里的 db.database —— 那个键从来没人写入，
	// dsn 恒为空字符串，于是打开一个空库，必然报 "no such table: user"。
	dbFile := resolvePanelDBFile()
	if _, err := os.Stat(dbFile); err != nil {
		log.Fatalf("数据库文件不存在: %s (%v)\n请确认 conf.yaml 里的 system.base_dir 是否正确，或用 -c 指定正确的配置文件", dbFile, err)
	}
	fmt.Printf("数据库: %s\n", dbFile)

	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	user, err := pickUserToReset(db, strings.TrimSpace(firstPositionalArg()))
	if err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Printf("即将重置账号: %s (role=%s)\n", user.Email, user.Role)
	// 状态字段与登录校验有关，停用状态下改了密码照样登不进去，提前说清楚
	if user.Status != 20 {
		fmt.Printf("警告: 该账号当前 status=%d（正常为 20），重置密码后可能仍无法登录\n", user.Status)
	}

	newPassword, err := readNewPassword()
	if err != nil {
		log.Fatalf("%v", err)
	}

	encoded, err := cryptx.EncodePassword(newPassword)
	if err != nil {
		log.Fatalf("密码加密失败，未做任何修改: %v", err)
	}

	if err := db.Table("user").Where("id = ?", user.ID).Update("password", encoded).Error; err != nil {
		log.Fatalf("更新密码失败: %v", err)
	}

	// 注意：不回显新密码，避免进入终端记录 / CI 日志
	fmt.Println("-----------------------------------------")
	fmt.Println("密码重置成功，请用新密码登录。")
	fmt.Printf("账号: %s\n", user.Email)
	fmt.Println("提示: 旧的登录会话不会自动失效，建议重启面板（gpc panel restart）")
	fmt.Println("-----------------------------------------")
}

// resolvePanelDBFile 推导面板数据库路径，解析顺序与 init/conf 保持一致：
// 环境变量 > conf.yaml 的 system.base_dir > 平台默认值。
func resolvePanelDBFile() string {
	baseDir := strings.TrimSpace(os.Getenv("GOPANEL_BASE_DIR"))
	if baseDir == "" {
		baseDir = strings.TrimSpace(os.Getenv("GPC_BASE_DIR"))
	}
	if baseDir == "" {
		baseDir = strings.TrimSpace(viper.GetString("system.base_dir"))
	}
	if baseDir == "" {
		baseDir = "/opt/gopanel"
		if runtime.GOOS != "linux" || os.Geteuid() != 0 {
			if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
				baseDir = filepath.Join(homeDir, ".gopanel")
			}
		}
	}
	// init/conf 里 DbPath 也是无条件 base_dir/db，这里不读 system.db_path，保持单一来源
	return filepath.Join(filepath.Clean(baseDir), "db", "gopanel.db")
}

// firstPositionalArg 取第一个非选项参数，用于 `gopanel --reset-password <邮箱>`
func firstPositionalArg() string {
	for _, arg := range pflag.Args() {
		if strings.TrimSpace(arg) != "" {
			return arg
		}
	}
	return ""
}

// pickUserToReset 定位要重置的账号：给了邮箱就按邮箱找，否则找超级管理员。
// 原实现额外限定了 status = 20，超管一旦被停用就直接"查不到用户"，这里放宽并改为事后提示。
func pickUserToReset(db *gorm.DB, email string) (*model.User, error) {
	query := db.Table("user")
	if email != "" {
		query = query.Where("email = ?", email)
	} else {
		query = query.Where("role = ?", "SUPER")
	}

	var users []model.User
	if err := query.Order("id").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}

	if len(users) == 0 {
		if email != "" {
			return nil, fmt.Errorf("没有找到邮箱为 %s 的账号\n%s", email, describeAccounts(db))
		}
		return nil, fmt.Errorf("没有找到超级管理员账号\n%s", describeAccounts(db))
	}
	if len(users) > 1 {
		var lines []string
		for _, u := range users {
			lines = append(lines, fmt.Sprintf("  - %s (role=%s, status=%d)", u.Email, u.Role, u.Status))
		}
		return nil, fmt.Errorf("匹配到多个账号，请显式指定邮箱：gopanel --reset-password <邮箱>\n%s", strings.Join(lines, "\n"))
	}
	return &users[0], nil
}

// describeAccounts 在找不到目标账号时，把库里现有账号列出来帮助定位
func describeAccounts(db *gorm.DB) string {
	var users []model.User
	if err := db.Table("user").Order("id").Find(&users).Error; err != nil || len(users) == 0 {
		return "当前数据库中没有任何账号"
	}
	lines := []string{"当前数据库中的账号："}
	for _, u := range users {
		lines = append(lines, fmt.Sprintf("  - %s (role=%s, status=%d)", u.Email, u.Role, u.Status))
	}
	return strings.Join(lines, "\n")
}

// readNewPassword 读取新密码。
// 交互式终端：隐藏输入 + 二次确认；非交互（管道）：读一整行。
// 原实现用 fmt.Scan，遇到空格会静默截断（"My Pass" 只存到 "My"），是最难查的一类坑。
func readNewPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Print("请输入新密码（输入内容不显示）: ")
		first, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("读取密码失败: %v", err)
		}
		fmt.Print("请再次输入新密码: ")
		second, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("读取密码失败: %v", err)
		}
		if string(first) != string(second) {
			return "", errors.New("两次输入的密码不一致，未做任何修改")
		}
		return validateNewPassword(string(first))
	}

	// 非交互场景（如 echo 'pw' | gopanel --reset-password），整行读取，保留空格
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("读取密码失败: %v", err)
	}
	return validateNewPassword(strings.TrimRight(line, "\r\n"))
}

func validateNewPassword(pw string) (string, error) {
	if strings.TrimSpace(pw) == "" {
		return "", errors.New("密码不能为空，未做任何修改")
	}
	if len(pw) < 6 {
		return "", errors.New("密码长度至少 6 位，未做任何修改")
	}
	// bcrypt 只处理前 72 字节，超长会直接报错（x/crypto: ErrPasswordTooLong）
	if len(pw) > 72 {
		return "", errors.New("密码长度不能超过 72 字节（bcrypt 限制），未做任何修改")
	}
	return pw, nil
}
