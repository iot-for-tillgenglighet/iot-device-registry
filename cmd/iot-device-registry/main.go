package main

import (
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/database"
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/handler"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
	log "github.com/sirupsen/logrus"
)

func main() {

	serviceName := "iot-device-registry"

	log.Infof("Starting up %s ...", serviceName)

	config := messaging.LoadConfiguration(serviceName)
	messenger, _ := messaging.Initialize(config)

	defer messenger.Close()

	db, _ := database.NewDatabaseConnection()

	handler.CreateRouterAndStartServing(messenger, db)
}
