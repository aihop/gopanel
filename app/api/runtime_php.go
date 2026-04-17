package api

import (
	"encoding/json"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/gofiber/fiber/v3"
)

func RuntimePhpExtensionsSearch(c fiber.Ctx) error {
	_, err := e.BodyToStruct[request.RuntimePHPExtensionsSearch](c.Body())
	if err != nil {
		return c.JSON(e.Result(err))
	}
	dataJson := `[
            {
                "id": 1,
                "createdAt": "2025-07-16T06:35:44.890502632Z",
                "updatedAt": "2025-07-16T06:35:44.890502632Z",
                "name": "Default",
                "extensions": "bcmath,ftp,gd,gettext,intl,mysqli,pcntl,pdo_mysql,shmop,soap,sockets,sysvsem,xmlrpc,zip"
            },
            {
                "id": 2,
                "createdAt": "2025-07-16T06:35:44.891640198Z",
                "updatedAt": "2025-07-16T06:35:44.891640198Z",
                "name": "WordPress",
                "extensions": "exif,igbinary,imagick,intl,zip,apcu,memcached,opcache,redis,shmop,mysqli,pdo_mysql,gd"
            },
            {
                "id": 3,
                "createdAt": "2025-07-16T06:35:44.893683038Z",
                "updatedAt": "2025-07-16T06:35:44.893683038Z",
                "name": "Flarum",
                "extensions": "curl,gd,pdo_mysql,mysqli,bz2,exif,yaf,imap"
            },
            {
                "id": 4,
                "createdAt": "2025-07-16T06:35:44.894832613Z",
                "updatedAt": "2025-07-16T06:35:44.894832613Z",
                "name": "SeaCMS",
                "extensions": "mysqli,pdo_mysql,gd,curl"
            },
            {
                "id": 5,
                "createdAt": "2025-07-16T06:35:44.89635395Z",
                "updatedAt": "2025-07-16T06:35:44.89635395Z",
                "name": "Dev",
                "extensions": "bcmath,ftp,gd,gettext,intl,mysqli,pcntl,pdo_mysql,shmop,soap,sockets,sysvsem,xmlrpc,zip,exif,igbinary,imagick,apcu,memcached,opcache,redis,bc,image,dom,iconv,mbstring,mysqlnd,openssl,pdo,tokenizer,xml,curl,bz2,yaf,imap,xdebug,swoole,pdo_pgsql,fileinfo,pgsql,calendar,gmp"
            }
        ]`
	data := []map[string]interface{}{}
	if err := json.Unmarshal([]byte(dataJson), &data); err != nil {
		return c.JSON(e.Result(err))
	}
	return c.JSON(e.Succ(data))
}
