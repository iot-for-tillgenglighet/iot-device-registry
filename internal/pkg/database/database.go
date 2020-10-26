package database

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/models"
	log "github.com/sirupsen/logrus"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

//Datastore is an interface that is used to inject the database into different handlers to improve testability
type Datastore interface {
}

var dbCtxKey = &databaseContextKey{"database"}

type databaseContextKey struct {
	name string
}

// Middleware packs a pointer to the datastore into context
func Middleware(db Datastore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), dbCtxKey, db)

			// and call the next with our new context
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

//GetFromContext extracts the database wrapper, if any, from the provided context
func GetFromContext(ctx context.Context) (Datastore, error) {
	db, ok := ctx.Value(dbCtxKey).(Datastore)
	if ok {
		return db, nil
	}

	return nil, errors.New("Failed to decode database from context")
}

type myDB struct {
	impl *gorm.DB
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

//NewDatabaseConnection initializes a new connection to the database and wraps it in a Datastore
func NewDatabaseConnection() (Datastore, error) {
	db := &myDB{}

	dbHost := os.Getenv("DEVREG_DB_HOST")
	username := os.Getenv("DEVREG_DB_USER")
	dbName := os.Getenv("DEVREG_DB_NAME")
	password := os.Getenv("DEVREG_DB_PASSWORD")
	sslMode := getEnv("DEVREG_DB_SSLMODE", "require")

	dbURI := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s password=%s", dbHost, username, dbName, sslMode, password)

	for {
		log.Printf("Connecting to database host %s ...\n", dbHost)
		conn, err := gorm.Open("postgres", dbURI)
		if err != nil {
			log.Fatalf("Failed to connect to database %s \n", err)
			time.Sleep(3 * time.Second)
		} else {
			db.impl = conn
			db.impl.Debug().AutoMigrate(&models.Device{})
			break
		}
		defer conn.Close()
	}

	return db, nil
}
