package geo

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/aihop/gopanel/global"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

var (
	once    sync.Once
	initErr error
	IP2     *xdb.Searcher
)

type GeoInfo struct {
	Region      string
	CountryCode string
	Country     string
	State       string
	City        string
}

func Init() error {
	once.Do(func() {
		data, err := global.EmbedFS.ReadFile("resource/geo/region.xdb")
		if err != nil {
			initErr = fmt.Errorf("read xdb file failed: %w", err)
			return
		}
		searcher, err := xdb.NewWithBuffer(xdb.IPv4, data)
		if err != nil {
			initErr = fmt.Errorf("init ip2region buffer failed: %w", err)
			return
		}
		IP2 = searcher

		log.Println("GEO database init success")
	})
	return initErr
}

func Region(ip string) string {
	if IP2 == nil || ip == "" {
		return ""
	}
	if ip == "" {
		return ""
	}
	res, _ := IP2.Search(ip)
	return format(res)
}

func format(re string) string {
	arr := strings.Split(re, "|")
	var newArr []string
	for _, v := range arr {
		if v != "0" && v != "" && v != " " {
			newArr = append(newArr, v)
		}
	}
	return strings.Join(newArr, " ")
}
