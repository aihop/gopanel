package global

import (
	"embed"

	"github.com/aihop/gopanel/config"
	"github.com/aihop/gopanel/init/cache/badger_db"
	"github.com/aihop/gopanel/init/session/psession"
	"github.com/aihop/gopanel/pkg/zlog"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-playground/validator/v10"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var (
	DB        *gorm.DB
	MonitorDB *gorm.DB
	LOG       *zlog.Logger
	VALID     *validator.Validate
	SESSION   *psession.PSession
	CACHE     *badger_db.Cache
	CacheDb   *badger.DB
	Viper     *viper.Viper

	Cron          *cron.Cron
	MonitorCronID cron.EntryID

	I18n       *i18n.Localizer
	I18nForCmd *i18n.Localizer
	CONF       config.ServerConfig
	EmbedFS    embed.FS
)
