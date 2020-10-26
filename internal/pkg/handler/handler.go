package handler

import (
	"compress/flate"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/database"
	gql "github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/graphql"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging/telemetry"

	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"

	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"
	ngsi "github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/ngsi-ld"
)

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
	router.Get("/ngsi-ld/v1/entities", ngsi.NewQueryEntitiesHandler(contextRegistry))
	router.Patch("/ngsi-ld/v1/entities/{entity}/attrs/", ngsi.NewUpdateEntityAttributesHandler(contextRegistry))
	router.Post("/ngsi-ld/v1/entities", ngsi.NewCreateEntityHandler(contextRegistry))
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

	return router
}

//MessagingContext is an interface that allows mocking of messaging.Context parameters
type MessagingContext interface {
	PublishOnTopic(message messaging.TopicMessage) error
}

func createContextRegistry(messenger MessagingContext) ngsi.ContextRegistry {
	contextRegistry := ngsi.NewContextRegistry()
	ctxSource := contextSource{messenger: messenger}
	contextRegistry.Register(&ctxSource)
	return contextRegistry
}

//CreateRouterAndStartServing sets up the NGSI-LD router and starts serving incoming requests
func CreateRouterAndStartServing(messenger MessagingContext, db database.Datastore) {
	contextRegistry := createContextRegistry(messenger)
	router := createRequestRouter(contextRegistry)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8880"
	}

	log.Printf("Starting iot-device-registry on port %s.\n", port)

	log.Fatal(http.ListenAndServe(":"+port, router.impl))
}

type contextSource struct {
	messenger MessagingContext
	devices   []fiware.Device
}

func (cs contextSource) ProvidesEntitiesWithMatchingID(entityID string) bool {
	return strings.HasPrefix(entityID, "urn:ngsi-ld:Device:")
}

func (cs *contextSource) CreateEntity(typeName, entityID string, req ngsi.Request) error {
	device := &fiware.Device{}
	err := req.DecodeBodyInto(device)

	// TODO: Save this device somewhere ...
	log.Warningln("Creating new Device entities is not implemented yet!")

	return err
}

func (cs *contextSource) GetEntities(query ngsi.Query, callback ngsi.QueryEntitiesCallback) error {

	var err error

	for _, device := range cs.devices {
		err = callback(device)
		if err != nil {
			break
		}
	}

	return err
}

func (cs contextSource) ProvidesAttribute(attributeName string) bool {
	return attributeName == "value"
}

func (cs contextSource) ProvidesType(typeName string) bool {
	return typeName == "Device"
}

func (cs *contextSource) UpdateEntityAttributes(entityID string, req ngsi.Request) error {

	updateSource := &fiware.Device{}
	err := req.DecodeBodyInto(updateSource)
	if err != nil {
		log.Errorln("Failed to decode PATCH body in UpdateEntityAttributes: " + err.Error())
		return err
	}

	// Truncate the leading "urn:ngsi-ld:Device:" from the device id string
	shortEntityID := entityID[19:]

	postWaterTempTelemetryIfDeviceIsAWaterTempDevice(cs.messenger, shortEntityID, updateSource.Value.Value)

	for index, device := range cs.devices {
		if device.ID == entityID {
			cs.devices[index].Value.Value = updateSource.Value.Value
			return nil
		}
	}

	cs.devices = append(cs.devices, *fiware.NewDevice(shortEntityID, updateSource.Value.Value))

	return nil
}

//This is a hack to decode the value and send it as a telemetry message over RabbitMQ for PoC purposes.
func postWaterTempTelemetryIfDeviceIsAWaterTempDevice(messenger MessagingContext, device, value string) {
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
						messenger.PublishOnTopic(
							telemetry.NewWaterTemperatureTelemetry(temp, device, 0.0, 0.0),
						)
					}
					return
				}
			}
		}
	}
}
