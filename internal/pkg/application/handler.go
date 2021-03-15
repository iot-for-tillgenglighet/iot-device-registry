package application

import (
	"compress/flate"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	gql "github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/_presentation/api/graphql"
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/repositories/database"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging/telemetry"

	"github.com/rs/cors"

	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"
	ngsi "github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/ngsi-ld"
	ngsitypes "github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/ngsi-ld/types"
)

//RequestRouter needs a comment
type RequestRouter struct {
	impl *chi.Mux
}

func (router *RequestRouter) addGraphQLHandlers() {
	gqlServer := handler.New(gql.NewExecutableSchema(gql.Config{Resolvers: &gql.Resolver{}}))
	gqlServer.AddTransport(&transport.POST{})
	gqlServer.Use(extension.Introspection{})

	router.impl.Handle("/api/graphql/playground", playground.Handler("GraphQL playground", "/api/graphql"))
	router.impl.Handle("/api/graphql", gqlServer)
}

func (router *RequestRouter) addNGSIHandlers(contextRegistry ngsi.ContextRegistry) {
	router.Get("/ngsi-ld/v1/entities/{entity}", ngsi.NewRetrieveEntityHandler(contextRegistry))
	router.Get("/ngsi-ld/v1/entities", ngsi.NewQueryEntitiesHandler(contextRegistry))
	router.Patch("/ngsi-ld/v1/entities/{entity}/attrs/", ngsi.NewUpdateEntityAttributesHandler(contextRegistry))
	router.Post("/ngsi-ld/v1/entities", ngsi.NewCreateEntityHandler(contextRegistry))
}

func (router *RequestRouter) addProbeHandlers() {
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

//Get accepts a pattern that should be routed to the handlerFn on a GET request
func (router *RequestRouter) Get(pattern string, handlerFn http.HandlerFunc) {
	router.impl.Get(pattern, handlerFn)
}

//Patch accepts a pattern that should be routed to the handlerFn on a PATCH request
func (router *RequestRouter) Patch(pattern string, handlerFn http.HandlerFunc) {
	router.impl.Patch(pattern, handlerFn)
}

//Post accepts a pattern that should be routed to the handlerFn on a POST request
func (router *RequestRouter) Post(pattern string, handlerFn http.HandlerFunc) {
	router.impl.Post(pattern, handlerFn)
}

func newRequestRouter() *RequestRouter {
	router := &RequestRouter{impl: chi.NewRouter()}

	router.impl.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	// Enable gzip compression for ngsi-ld responses
	compressor := middleware.NewCompressor(flate.DefaultCompression, "application/json", "application/ld+json")
	router.impl.Use(compressor.Handler)
	router.impl.Use(middleware.Logger)

	return router
}

func createRequestRouter(contextRegistry ngsi.ContextRegistry) *RequestRouter {
	router := newRequestRouter()

	router.addGraphQLHandlers()
	router.addNGSIHandlers(contextRegistry)
	router.addProbeHandlers()

	return router
}

//MessagingContext is an interface that allows mocking of messaging.Context parameters
type MessagingContext interface {
	PublishOnTopic(message messaging.TopicMessage) error
}

func createContextRegistry(log logging.Logger, messenger MessagingContext, db database.Datastore) ngsi.ContextRegistry {
	contextRegistry := ngsi.NewContextRegistry()
	ctxSource := contextSource{db: db, log: log, messenger: messenger}
	contextRegistry.Register(&ctxSource)
	return contextRegistry
}

//CreateRouterAndStartServing sets up the NGSI-LD router and starts serving incoming requests
func CreateRouterAndStartServing(log logging.Logger, messenger MessagingContext, db database.Datastore) {
	contextRegistry := createContextRegistry(log, messenger, db)
	router := createRequestRouter(contextRegistry)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8880"
	}

	log.Infof("Starting iot-device-registry on port %s.\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router.impl))
}

type contextSource struct {
	db        database.Datastore
	log       logging.Logger
	messenger MessagingContext
}

func (cs contextSource) ProvidesEntitiesWithMatchingID(entityID string) bool {
	if strings.Contains(entityID, "DeviceModel") {
		return strings.HasPrefix(entityID, fiware.DeviceModelIDPrefix)
	}
	return strings.HasPrefix(entityID, fiware.DeviceIDPrefix)
}

func (cs *contextSource) CreateEntity(typeName, entityID string, req ngsi.Request) error {
	var err error

	if typeName == "Device" {
		device := &fiware.Device{}
		err = req.DecodeBodyInto(device)
		if err != nil {
			cs.log.Errorf("Failed to decode body into Device: %s", err.Error())
			return err
		}

		_, err = cs.db.CreateDevice(device)

	} else if typeName == "DeviceModel" {
		deviceModel := &fiware.DeviceModel{}
		err = req.DecodeBodyInto(deviceModel)
		if err != nil {
			cs.log.Errorf("Failed to decode body into DeviceModel: %s", err.Error())
			return err
		}
		_, err = cs.db.CreateDeviceModel(deviceModel)

	} else {
		errorMessage := fmt.Sprintf("Entity of type  " + typeName + " is not supported.")
		cs.log.Errorf(errorMessage)
		return errors.New(errorMessage)
	}

	return err
}

func (cs *contextSource) GetEntities(query ngsi.Query, callback ngsi.QueryEntitiesCallback) error {

	var err error

	if query == nil {
		return errors.New("GetEntities: query may not be nil")
	}

	for _, typeName := range query.EntityTypes() {
		if typeName == "Device" {
			devices, err := cs.db.GetDevices()
			if err != nil {
				return fmt.Errorf("unable to get Device entities: %s", err.Error())
			}

			for _, device := range devices {
				fiwareDevice := fiware.NewDevice(device.DeviceID, url.QueryEscape(device.Value))
				deviceModel, err := cs.db.GetDeviceModelFromPrimaryKey(device.DeviceModelID)
				if err == nil {
					fiwareDevice.RefDeviceModel, _ = fiware.NewDeviceModelRelationship(deviceModel.DeviceModelID)
				}

				if !device.DateLastValueReported.IsZero() {
					fiwareDevice.DateLastValueReported = ngsitypes.CreateDateTimeProperty(
						device.DateLastValueReported.Format(time.RFC3339),
					)
				}

				err = callback(fiwareDevice)
				if err != nil {
					break
				}
			}
		} else if typeName == "DeviceModel" {
			deviceModels, err := cs.db.GetDeviceModels()
			if err != nil {
				return fmt.Errorf("unable to get DeviceModels: %s", err.Error())
			}

			for _, deviceModel := range deviceModels {
				fiwareDeviceModel := fiware.NewDeviceModel(deviceModel.DeviceModelID, []string{deviceModel.Category})
				fiwareDeviceModel.BrandName = ngsitypes.NewTextProperty(deviceModel.BrandName)
				fiwareDeviceModel.ModelName = ngsitypes.NewTextProperty(deviceModel.ModelName)
				fiwareDeviceModel.ManufacturerName = ngsitypes.NewTextProperty(deviceModel.ManufacturerName)
				fiwareDeviceModel.Name = ngsitypes.NewTextProperty(deviceModel.Name)

				err = callback(fiwareDeviceModel)
				if err != nil {
					break
				}
			}
		}
	}

	return err
}

func (cs contextSource) ProvidesAttribute(attributeName string) bool {
	return attributeName == "value"
}

func (cs contextSource) ProvidesType(typeName string) bool {
	return (typeName == "DeviceModel" || typeName == "Device")
}

func (cs *contextSource) RetrieveEntity(entityID string, req ngsi.Request) (ngsi.Entity, error) {
	if strings.HasPrefix(entityID, fiware.DeviceIDPrefix) {
		shortEntityID := entityID[len(fiware.DeviceIDPrefix):]

		device, err := cs.db.GetDeviceFromID(shortEntityID)
		if err != nil {
			return nil, fmt.Errorf("no Device found with ID %s: %s", shortEntityID, err.Error())
		}

		fiwareDevice := fiware.NewDevice(device.DeviceID, url.QueryEscape(device.Value))
		deviceModel, err := cs.db.GetDeviceModelFromPrimaryKey(device.DeviceModelID)
		if err != nil {
			return nil, fmt.Errorf("no valid DeviceModel found: %s", err.Error())
		}

		fiwareDevice.RefDeviceModel, _ = fiware.NewDeviceModelRelationship(deviceModel.DeviceModelID)
		if !device.DateLastValueReported.IsZero() {
			fiwareDevice.DateLastValueReported = ngsitypes.CreateDateTimeProperty(
				device.DateLastValueReported.Format(time.RFC3339),
			)
		}

		return fiwareDevice, nil
	} else if strings.HasPrefix(entityID, fiware.DeviceModelIDPrefix) {
		shortEntityID := entityID[len(fiware.DeviceModelIDPrefix):]

		deviceModel, err := cs.db.GetDeviceModelFromID(shortEntityID)
		if err != nil {
			return nil, fmt.Errorf("no DeviceModel found with ID %s: %s", shortEntityID, err.Error())
		}

		fiwareDeviceModel := fiware.NewDeviceModel(deviceModel.DeviceModelID, []string{deviceModel.Category})
		fiwareDeviceModel.BrandName = ngsitypes.NewTextProperty(deviceModel.BrandName)
		fiwareDeviceModel.ModelName = ngsitypes.NewTextProperty(deviceModel.ModelName)
		fiwareDeviceModel.ManufacturerName = ngsitypes.NewTextProperty(deviceModel.ManufacturerName)
		fiwareDeviceModel.Name = ngsitypes.NewTextProperty(deviceModel.Name)

		return fiwareDeviceModel, nil
	}

	return nil, fmt.Errorf("unable to find entity type from entity ID: %s", entityID)
}

func (cs *contextSource) UpdateEntityAttributes(entityID string, req ngsi.Request) error {

	updateSource := &fiware.Device{}
	err := req.DecodeBodyInto(updateSource)
	if err != nil {
		cs.log.Errorf("Failed to decode PATCH body in UpdateEntityAttributes: %s", err.Error())
		return err
	}

	// Truncate the fiware prefix from the device id string
	shortEntityID := entityID[len(fiware.DeviceIDPrefix):]
	device, err := cs.db.GetDeviceFromID(shortEntityID)
	if err != nil {
		cs.log.Errorf("Unable to find device %s for attributes update.", entityID)
		return err
	}

	value, err := url.QueryUnescape(updateSource.Value.Value)
	if err == nil {
		err = cs.db.UpdateDeviceValue(shortEntityID, value)
		if err == nil {
			postWaterTempTelemetryIfDeviceIsAWaterTempDevice(
				cs,
				shortEntityID,
				device.Latitude, device.Longitude,
				value,
			)
		}
	}

	return err
}

//This is a hack to decode the value and send it as a telemetry message over RabbitMQ for PoC purposes.
func postWaterTempTelemetryIfDeviceIsAWaterTempDevice(cs *contextSource, device string, lat, lon float64, value string) {
	if strings.Contains(device, "sk-elt-temp-") {
		decodedValue, err := url.QueryUnescape(value)
		if err != nil {
			return
		}

		values := strings.Split(decodedValue, ";")
		for _, v := range values {
			parts := strings.Split(v, "=")
			if len(parts) == 2 {
				if parts[0] == "t" {
					temp, err := strconv.ParseFloat(parts[1], 64)
					if err == nil {
						// TODO: Make this configurable
						const MinTemp float64 = -0.5
						const MaxTemp float64 = 15.0
						if temp >= MinTemp && temp <= MaxTemp {
							cs.messenger.PublishOnTopic(
								telemetry.NewWaterTemperatureTelemetry(temp, device, lat, lon),
							)
						} else {
							cs.log.Infof(
								"ignored water temp value from %s: %f not in allowed range [%f,%f]",
								device, temp, MinTemp, MaxTemp,
							)
						}
					}
					return
				}
			}
		}
	}
}
