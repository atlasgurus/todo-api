package main

import (
	"github.com/tintash-training/todo-api/app"
	"github.com/tintash-training/todo-api/app/config"
)

func main() {
	config := config.GetConf()
	app := &app.App{}
	app.Start(config)
}
