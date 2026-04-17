package db

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/aihop/gopanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func Init() {

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	if err := os.MkdirAll(global.CONF.System.DbPath, 0o755); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}
	dbPath := filepath.Join(global.CONF.System.DbPath, "gopanel.db")
	db, err := openSQLite(dbPath)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	global.DB = db
	log.Println("Database connect success")

	fmt.Println(global.CONF.System.DbPath, "global.CONF.System.DbPath")

	initMonitorDB(newLogger)
}

func initMonitorDB(newLogger logger.Interface) {
	if _, err := os.Stat(global.CONF.System.DbPath); err != nil {
		if err := os.MkdirAll(global.CONF.System.DbPath, os.ModePerm); err != nil {
			panic(fmt.Errorf("init db dir failed, err: %v", err))
		}
	}
	fullPath := path.Join(global.CONF.System.DbPath, "monitor.db")
	if _, err := os.Stat(fullPath); err != nil {
		f, err := os.Create(fullPath)
		if err != nil {
			panic(fmt.Errorf("init db file failed, err: %v", err))
		}
		_ = f.Close()
	}

	db, err := gorm.Open(sqlite.Open(fullPath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   newLogger,
	})
	if err != nil {
		panic(err)
	}
	sqlDB, dbError := db.DB()
	if dbError != nil {
		panic(dbError)
	}
	sqlDB.SetConnMaxIdleTime(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	global.MonitorDB = db
	if global.LOG != nil {
		global.LOG.Info("init monitor db successfully")
	} else {
		log.Println("init monitor db successfully")
	}
}

func openSQLite(dbPath string) (*gorm.DB, error) {
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			PrepareStmt:                              true,  // 缓存预编译语句
			AllowGlobalUpdate:                        false, // 关闭无条件的全局更新（关闭可以防止全局更新删除）
			QueryFields:                              true,  // 查询 * 时，会自动填写所有字段名
			DisableForeignKeyConstraintWhenMigrating: true,  // 禁用外键约束
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true, // 使用单数表名
			},
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			return db, nil
		}
		lastErr = err
		if !isRecoverableSQLiteOpenError(err) {
			return nil, err
		}
		log.Printf("sqlite schema recovery triggered for %s (attempt %d): %v", dbPath, attempt+1, err)
		if repairErr := repairSQLiteSchema(dbPath, err); repairErr != nil {
			log.Printf("sqlite schema recovery failed for %s: %v", dbPath, repairErr)
			return nil, err
		}
	}
	return nil, lastErr
}
