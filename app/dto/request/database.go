package request

type DatabaseCreate struct {
	ServerID   uint   `json:"serverId" validate:"required"`
	Name       string `json:"name" validate:"required"`
	CreateUser bool   `json:"createUser"`
	Username   string `json:"username" validate:"required_if=CreateUser true"`
	Password   string `json:"password" validate:"required_if=CreateUser true"`
	Host       string `json:"host"`
	Comment    string `json:"comment"`
}

type DatabaseDelete struct {
	ServerID uint   `json:"serverId" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

type DatabaseComment struct {
	ServerID uint   `json:"serverId" validate:"required"`
	Name     string `json:"name" validate:"required"`
	Comment  string `json:"comment"`
}

type DatabaseServerCreate struct {
	Name     string `json:"name" validate:"required"`
	Type     string `json:"type" validate:"required,oneof=mysql postgresql redis sqlite"`
	Host     string `json:"host" validate:"required"` // 对于 SQLite，Host 是文件绝对路径
	Port     uint   `json:"port" validate:"required_unless=Type sqlite,gte=0,lte=65535"`
	Username string `json:"username" validate:"required_unless=Type redis Type sqlite"` // SQL 类型需用户名
	Password string `json:"password" validate:"required_unless=Type redis Type sqlite"` // SQL 类型需密码
	Remark   string `json:"remark"`
}

type DatabaseServerUpdate struct {
	ID       uint   `json:"id" validate:"required"`
	Name     string `json:"name" validate:"required"`
	Type     string `json:"type" validate:"required,oneof=mysql postgresql redis sqlite"`
	Host     string `json:"host" validate:"required"`
	Port     uint   `json:"port" validate:"required_unless=Type sqlite,gte=0,lte=65535"`
	Username string `json:"username" validate:"required_unless=Type redis Type sqlite"` // SQL 类型需用户名
	Password string `json:"password" validate:"required_unless=Type redis Type sqlite"` // SQL 类型需密码
	Remark   string `json:"remark"`
}

type DatabaseUserCreate struct {
	ServerID   uint     `json:"serverId" validate:"required"`
	Username   string   `json:"username" validate:"required"`
	Password   string   `json:"password" validate:"required"`
	Host       string   `json:"host"`
	Privileges []string `json:"privileges"`
	Remark     string   `json:"remark"`
}

type DatabaseUserUpdate struct {
	ID         uint     `json:"id"`
	ServerID   uint     `json:"serverId"`
	Username   string   `json:"username"`
	Host       string   `json:"host"`
	Password   string   `json:"password"`
	Privileges []string `json:"privileges"`
	Remark     string   `json:"remark"`
}

type DatabaseUserDelete struct {
	ID       uint   `json:"id"`
	ServerId uint   `json:"serverId"`
	Username string `json:"username"`
}

type DatabaseUserGet struct {
	ID       uint   `json:"id"`
	ServerID uint   `json:"serverId"`
	Username string `json:"username"`
	Host     string `json:"host"`
}
