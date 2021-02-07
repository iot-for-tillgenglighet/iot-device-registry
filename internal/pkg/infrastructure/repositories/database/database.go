package database

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/repositories/models"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//Datastore is an interface that is used to inject the database into different handlers to improve testability
type Datastore interface {
	CreateDevice(device *fiware.Device) (*models.Device, error)
	CreateDeviceModel(deviceModel *fiware.DeviceModel) (*models.DeviceModel, error)
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

	controlledProperties []models.DeviceControlledProperty
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

//ConnectorFunc is used to inject a database connection method into NewDatabaseConnection
type ConnectorFunc func() (*gorm.DB, error)

//NewPostgreSQLConnector opens a connection to a postgresql database
func NewPostgreSQLConnector(log logging.Logger) ConnectorFunc {
	dbHost := os.Getenv("DEVREG_DB_HOST")
	username := os.Getenv("DEVREG_DB_USER")
	dbName := os.Getenv("DEVREG_DB_NAME")
	password := os.Getenv("DEVREG_DB_PASSWORD")
	sslMode := getEnv("DEVREG_DB_SSLMODE", "require")

	dbURI := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s password=%s", dbHost, username, dbName, sslMode, password)

	return func() (*gorm.DB, error) {
		for {
			log.Infof("Connecting to database host %s ...\n", dbHost)
			db, err := gorm.Open(postgres.Open(dbURI), &gorm.Config{})
			if err != nil {
				log.Fatalf("Failed to connect to database %s\n", err)
				time.Sleep(3 * time.Second)
			} else {
				return db, nil
			}
		}
	}
}

//NewSQLiteConnector opens a connection to a local sqlite database
func NewSQLiteConnector() ConnectorFunc {
	return func() (*gorm.DB, error) {
		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})

		if err == nil {
			db.Exec("PRAGMA foreign_keys = ON")
		}

		return db, err
	}
}

//NewDatabaseConnection initializes a new connection to the database and wraps it in a Datastore
func NewDatabaseConnection(connect ConnectorFunc, log logging.Logger) (Datastore, error) {
	impl, err := connect()
	if err != nil {
		return nil, err
	}

	db := &myDB{
		impl: impl,
	}

	db.impl.Debug().AutoMigrate(&models.DeviceControlledProperty{})
	db.impl.Debug().AutoMigrate(&models.DeviceModel{})
	db.impl.Debug().AutoMigrate(&models.Device{})

	db.impl.Debug().Model(&models.DeviceModel{}).Association("DeviceControlledProperty")
	db.impl.Debug().Model(&models.Device{}).Association("DeviceModel")

	// Make sure that the controlled properties table is properly seeded
	props := map[string]string{
		"temperature": "t",
		"snowDepth":   "snow",
	}

	for property, abbreviation := range props {
		controlledProperty := models.DeviceControlledProperty{}

		result := db.impl.Where("name = ?", property).First(&controlledProperty)
		if result.RowsAffected == 0 {
			log.Infof("ControlledProperty %s not found in database. Creating ...", property)

			controlledProperty.Name = property
			controlledProperty.Abbreviation = abbreviation
			result = db.impl.Debug().Create(&controlledProperty)

			if result.Error != nil {
				log.Fatalf("Failed to seed DeviceControlledProperty into database %s", result.Error.Error())
				return nil, result.Error
			}
		}

		db.controlledProperties = append(db.controlledProperties, controlledProperty)
	}

	/*badtemp := models.DeviceModel{DeviceModelID: "urn:ngsi-ld:DeviceModel:badtemperatur", Category: "sensor"}
	badtemp.ControlledProperties = db.getControlledProperties("temperatur")

	livboj := models.DeviceModel{DeviceModelID: "urn:ngsi-ld:DeviceModel:livboj", Category: "sensor"}

	deviceModels := []models.DeviceModel{badtemp, livboj}

	for _, model := range deviceModels {
		m := models.DeviceModel{}
		result := db.impl.Debug().Where("device_model_id = ?", model.DeviceModelID).First(&m)
		if result.RowsAffected == 0 {
			m.DeviceModelID = model.DeviceModelID
			m.Category = model.Category
			m.ControlledProperties = model.ControlledProperties

			result = db.impl.Debug().Create(&m)
			if result.Error != nil {
				log.Fatalf("Failed to seed DeviceModel into database %s", result.Error.Error())
				return nil, result.Error
			}
		}
		db.deviceModels = append(db.deviceModels, m)
	}*/

	return db, nil
}

func (db *myDB) CreateDevice(src *fiware.Device) (*models.Device, error) {

	if src.RefDeviceModel == nil {
		return nil, fmt.Errorf("CreateDevice requires non-empty device model")
	}

	deviceModel, err := db.getDeviceModelFromString(src.RefDeviceModel.Object)
	if err != nil {
		return nil, err
	}

	device := &models.Device{
		DeviceID:    src.ID,
		DeviceModel: *deviceModel,
	}

	if src.Location != nil {
		device.Latitude = src.Location.Value.Coordinates[0]
		device.Longitude = src.Location.Value.Coordinates[1]
	}

	db.impl.Debug().Create(device)

	return device, nil
}

func (db *myDB) CreateDeviceModel(src *fiware.DeviceModel) (*models.DeviceModel, error) {

	deviceModel := &models.DeviceModel{
		DeviceModelID: src.ID,
		Category:      src.Category.Value[0],
	}

	db.impl.Debug().Create(deviceModel)

	return deviceModel, nil
}

func (db *myDB) getControlledProperties(properties ...string) []models.DeviceControlledProperty {
	found := []models.DeviceControlledProperty{}

	for _, p := range properties {
		for _, controlledProperty := range db.controlledProperties {
			if controlledProperty.Name == p {
				found = append(found, controlledProperty)
				break
			}
		}
	}

	return found
}

func (db *myDB) getDeviceModelFromString(deviceModelID string) (*models.DeviceModel, error) {
	m := &models.DeviceModel{}
	result := db.impl.Debug().Where("device_model_id = ?", deviceModelID).First(m)
	if result.RowsAffected == 1 {
		return m, nil
	}

	return nil, errors.New("No DeviceModel found matching " + deviceModelID)
}
