package main

import (
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/application"
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/repositories/database"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
)

func main() {

	serviceName := "iot-device-registry"

	log := logging.NewLogger()
	log.Infof("Starting up %s ...", serviceName)

	config := messaging.LoadConfiguration(serviceName)
	messenger, _ := messaging.Initialize(config)

	defer messenger.Close()

	db, _ := database.NewDatabaseConnection(database.NewPostgreSQLConnector(log), log)
	application.CreateRouterAndStartServing(log, messenger, db)
}
