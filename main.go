//go:build !desktop

package main

import (
	"embed"
	"log"

	"github.com/aihop/gopanel/app"
	"github.com/aihop/gopanel/global"
)

//go:embed public/* resource/*
var EmbedFS embed.FS

func main() {
	global.EmbedFS = EmbedFS
	app := app.App{}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
