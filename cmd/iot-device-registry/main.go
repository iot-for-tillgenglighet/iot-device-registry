package main

import (
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/handler"
	log "github.com/sirupsen/logrus"
)

func main() {

	serviceName := "iot-device-registry"

	log.Infof("Starting up %s ...", serviceName)

	handler.Router()
}
