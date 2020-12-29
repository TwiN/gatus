package main

import (
	"log"
	"os"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/controller"
	"github.com/TwinProduction/gatus/storage"
	"github.com/TwinProduction/gatus/watchdog"
)

func main() {
	cfg := loadConfiguration()
	database, err := storage.NewDatabase(*cfg.Storage)
	if err != nil {
		log.Fatalf("unable to construct database: %s", err)
	}
	go watchdog.Monitor(cfg, database)
	controller.Handle(database)
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
