package main

import (
	"os"
	"strconv"

	"github.com/ikolcov/microblog/internal/app"
)

func getServerPort() uint16 {
	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		if port, err := strconv.ParseUint(serverPort, 10, 16); err == nil {
			return uint16(port)
		}
	}
	panic("Port should be set in env var SERVER_PORT")
}

func getStorageMode() app.StorageMode {
	switch os.Getenv("STORAGE_MODE") {
	case "inmemory":
		return app.InMemory
	case "mongo":
		return app.Mongo
	case "cached":
		return app.Mongo
	}
	panic("Storage mode should be set in env var STORAGE_MODE")
}

func main() {
	config := app.AppConfig{
		Port:        getServerPort(),
		Mode:        getStorageMode(),
		MongoUrl:    os.Getenv("MONGO_URL"),
		MongoDbName: os.Getenv("MONGO_DBNAME"),
	}

	app.New(config).Start()
}
