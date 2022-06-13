package main

import (
	"flag"
	"github.com/tintash-training/todo-api/app"
	"github.com/tintash-training/todo-api/app/config"
)

func main() {
	// Default to INFO level for stderr. May override on command line e.g. -stderrthreshold=WARNING
	flag.Set("stderrthreshold", "INFO")
	flag.Parse()
	config := config.GetConf()
	app := &app.App{}
	app.Start(config)
}
