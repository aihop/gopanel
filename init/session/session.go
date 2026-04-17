package session

import (
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/session/psession"
)

func Init() {
	global.SESSION = psession.NewPSession(global.CACHE)
	global.LOG.Info("init session successfully")
}
