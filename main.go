package main

import (
	"embed"

	"github.com/aihop/gopanel/app"
	"github.com/aihop/gopanel/global"
)

//go:embed public/* resource/*
var EmbedFS embed.FS

func main() {
	global.EmbedFS = EmbedFS
	app := app.App{}
	app.Run()
}
