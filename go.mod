module github.com/iot-for-tillgenglighet/iot-device-registry

go 1.15

require (
	github.com/99designs/gqlgen v0.11.1
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/iot-for-tillgenglighet/messaging-golang v0.0.0-20201009211140-579335ad3c49
	github.com/iot-for-tillgenglighet/ngsi-ld-golang v0.0.0-20210203142601-3118f5101a69
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.7.0
	github.com/vektah/gqlparser/v2 v2.0.1
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	gorm.io/driver/postgres v1.0.5
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.20.7
)
