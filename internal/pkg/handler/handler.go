package handler

import (
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	gql "github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/graphql"

	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

func Router() {

	router := chi.NewRouter()

	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            true,
	}).Handler)

	srv := handler.New(gql.NewExecutableSchema(gql.Config{Resolvers: &gql.Resolver{}}))
	srv.AddTransport(&transport.POST{})
	srv.Use(extension.Introspection{})

	router.Handle("/api/graphql/playground", playground.Handler("GraphQL playground", "/api/graphql"))
	router.Handle("/api/graphql", srv)

	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8880"
	}

	log.Printf("Starting iot-device-registry on port %s.\n", port)

	log.Fatal(http.ListenAndServe(":"+port, router))
}
