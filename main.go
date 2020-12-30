package main

import (
	"os"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/controller"
	"github.com/TwinProduction/gatus/watchdog"
)

func main() {
	cfg := loadConfiguration()
	go watchdog.Monitor(cfg)
	controller.Handle()
}

func loadConfiguration() *config.Config {
	var err error
	customConfigFile := os.Getenv("GATUS_CONFIG_FILE")
	if len(customConfigFile) > 0 {
		err = config.Load(customConfigFile)
	} else {
		err = config.LoadDefaultConfiguration()
	}
	if err != nil {
		panic(err)
	}
	return config.Get()
}
