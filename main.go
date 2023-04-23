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

func main() {
	config := app.AppConfig{
		Port: getServerPort(),
	}

	app.New(config).Start()
}
