package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"slices"

	"github.com/aihop/gopanel/pkg/systemctl"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Postgres struct {
	db       *sql.DB
	username string
	password string
	address  string
	port     uint
}

func NewPostgres(username, password, address string, port uint) (*Postgres, error) {
	escapedUsername := url.QueryEscape(username)
	escapedPassword := url.QueryEscape(password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable", escapedUsername, escapedPassword, address, port)
	if password == "" {
		if username == "" {
			username = "postgres"
		}
		escapedUsername = url.QueryEscape(username)
		dsn = fmt.Sprintf("postgres://%s@%s:%d/postgres?sslmode=disable", escapedUsername, address, port)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("init postgres connection failed: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("connect to postgres failed: %w", err)
	}
	return &Postgres{
		db:       db,
		username: username,
		password: password,
		address:  address,
		port:     port,
	}, nil
}

func (r *Postgres) Close() error {
	return r.db.Close()
}

func (r *Postgres) Ping() error {
	return r.db.Ping()
}

func (r *Postgres) Query(query string, args ...any) (*sql.Rows, error) {
	return r.db.Query(query, args...)
}

func (r *Postgres) QueryRow(query string, args ...any) *sql.Row {
	return r.db.QueryRow(query, args...)
}

func (r *Postgres) Exec(query string, args ...any) (sql.Result, error) {
	return r.db.Exec(query, args...)
}

func (r *Postgres) Prepare(query string) (*sql.Stmt, error) {
	return r.db.Prepare(query)
}

func (r *Postgres) DatabaseCreate(name string) error {
	// postgres 不支持 CREATE DATABASE IF NOT EXISTS，但是为了保持与 MySQL 一致，先检查数据库是否存在
	exist, err := r.DatabaseExist(name)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	_, err = r.Exec(fmt.Sprintf("CREATE DATABASE %s", name))
	return err
}

func (r *Postgres) DatabaseDrop(name string) error {
	_, err := r.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
	return err
}

func (r *Postgres) DatabaseExist(name string) (bool, error) {
	var count int
	if err := r.QueryRow("SELECT COUNT(*) FROM pg_database WHERE datname = $1", name).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Postgres) DatabaseSize(name string) (int64, error) {
	query := fmt.Sprintf("SELECT pg_database_size('%s')", name)
	var size int64
	if err := r.QueryRow(query).Scan(&size); err != nil {
		return 0, err
	}
	return size, nil
}

func (r *Postgres) DatabaseComment(name, comment string) error {
	_, err := r.Exec(fmt.Sprintf("COMMENT ON DATABASE %s IS '%s'", name, comment))
	return err
}

func (r *Postgres) UserCreate(user, password string) error {
	_, err := r.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", user, password))
	if err != nil {
		return err
	}

	return nil
}

func (r *Postgres) UserDrop(user string) error {
	_, err := r.Exec(fmt.Sprintf("DROP USER IF EXISTS %s", user))
	if err != nil {
		return err
	}

	return systemctl.Reload("postgresql")
}

func (r *Postgres) UserPassword(user, password string) error {
	_, err := r.Exec(fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s'", user, password))
	return err
}

func (r *Postgres) UserPrivileges(user string) ([]string, error) {
	query := `
        SELECT d.datname
        FROM pg_catalog.pg_database d
        JOIN pg_catalog.pg_roles r ON d.datdba = r.oid
        WHERE r.rolname = $1
        AND d.datistemplate = false
        AND d.datname NOT IN ('template0', 'template1', 'postgres')
        ORDER BY d.datname;
    `

	rows, err := r.Query(query, user)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var databases []string

	for rows.Next() {
		var dbName string
		if err = rows.Scan(&dbName); err != nil {
			return nil, err
		}
		databases = append(databases, dbName)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return databases, nil
}

func (r *Postgres) PrivilegesGrant(user, database string) error {
	if _, err := r.Exec(fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", database, user)); err != nil {
		return err
	}
	if _, err := r.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", database, user)); err != nil {
		return err
	}

	return nil
}

func (r *Postgres) PrivilegesRevoke(user, database string) error {
	_, err := r.Exec(fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM %s", database, user))
	return err
}

func (r *Postgres) Users() ([]PostgresUser, error) {
	query := `
        SELECT rolname,
               rolsuper,
               rolcreaterole,
               rolcreatedb,
               rolreplication,
               rolbypassrls
        FROM pg_roles
        WHERE rolcanlogin = true;
    `
	rows, err := r.Query(query)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var users []PostgresUser
	for rows.Next() {
		var user PostgresUser
		var super, canCreateRole, canCreateDb, replication, bypassRls bool
		if err = rows.Scan(&user.Role, &super, &canCreateRole, &canCreateDb, &replication, &bypassRls); err != nil {
			return nil, err
		}

		permissions := map[string]bool{
			"Super":       super,
			"CreateRole":  canCreateRole,
			"CreateDB":    canCreateDb,
			"Replication": replication,
			"BypassRLS":   bypassRls,
		}
		for perm, enabled := range permissions {
			if enabled {
				user.Attributes = append(user.Attributes, perm)
			}
		}

		if len(user.Attributes) == 0 {
			user.Attributes = append(user.Attributes, "无")
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Postgres) Databases() ([]PostgresDatabase, error) {
	query := `
        SELECT 
            d.datname, 
            pg_catalog.pg_get_userbyid(d.datdba), 
            pg_catalog.pg_encoding_to_char(d.encoding),
            COALESCE(pg_catalog.shobj_description(d.oid, 'pg_database'), '')
        FROM pg_catalog.pg_database d
        WHERE datistemplate = false;
    `
	rows, err := r.Query(query)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var databases []PostgresDatabase
	for rows.Next() {
		var db PostgresDatabase
		if err := rows.Scan(&db.Name, &db.Owner, &db.Encoding, &db.Comment); err != nil {
			return nil, err
		}
		if slices.Contains([]string{"template0", "template1", "postgres"}, db.Name) {
			continue
		}
		databases = append(databases, db)
	}

	return databases, nil
}

type PostgresUser struct {
	Role       string   `json:"role"`
	Attributes []string `json:"attributes"`
}

type PostgresDatabase struct {
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	Encoding string `json:"encoding"`
	Comment  string `json:"comment"`
}
