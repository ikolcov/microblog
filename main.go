package main

import (
	"os"
	"strconv"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/log"
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

func encodeTaskFunc(data, password string) (string, error) {
	return "", nil
}

func startServer(redisUrl string) (*machinery.Server, error) {
	cnf := &config.Config{
		DefaultQueue:    "machinery_tasks",
		ResultsExpireIn: 3600,
		Broker:          "redis://" + redisUrl,
		ResultBackend:   "redis://" + redisUrl,
		Redis: &config.RedisConfig{
			MaxIdle:                3,
			IdleTimeout:            240,
			ReadTimeout:            15,
			WriteTimeout:           15,
			ConnectTimeout:         15,
			NormalTasksPollPeriod:  1000,
			DelayedTasksPollPeriod: 500,
		},
	}

	server, err := machinery.NewServer(cnf)
	if err != nil {
		return nil, err
	}

	// Register tasks
	tasks := map[string]interface{}{
		"encode": encodeTaskFunc,
	}

	return server, server.RegisterTasks(tasks)
}

func worker(server *machinery.Server) error {
	consumerTag := "machinery_worker"

	worker := server.NewWorker(consumerTag, 0)

	errorhandler := func(err error) {
		log.ERROR.Println("Something went wrong:", err)
	}

	worker.SetErrorHandler(errorhandler)

	return worker.Launch()
}

func main() {
	mongoUrl := os.Getenv("MONGO_URL")
	mongoDbName := os.Getenv("MONGO_DBNAME")
	redisUrl := os.Getenv("REDIS_URL")

	machineryServer, err := startServer(redisUrl)
	if err != nil {
		panic(err)
	}

	switch os.Getenv("APP_MODE") {
	case "SERVER":
		appConfig := app.AppConfig{
			Port:        getServerPort(),
			MongoUrl:    mongoUrl,
			MongoDbName: mongoDbName,
			RedisUrl:    redisUrl,
		}

		app.New(appConfig, machineryServer).Start()
	case "WORKER":
		worker(machineryServer)
	default:
		panic("APP_MODE must be either SERVER or WORKER")
	}
}
