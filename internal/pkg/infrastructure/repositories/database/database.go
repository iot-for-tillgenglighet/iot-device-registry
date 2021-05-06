package database

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
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
	GetDeviceFromID(id string) (*models.Device, error)
	GetDevices() ([]models.Device, error)
	GetDeviceModels() ([]models.DeviceModel, error)
	GetDeviceModelFromID(id string) (*models.DeviceModel, error)
	GetDeviceModelFromPrimaryKey(id uint) (*models.DeviceModel, error)
	UpdateDeviceValue(deviceID, value string) error
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

	return nil, errors.New("failed to decode database from context")
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
	dbHost := os.Getenv("DIWISE_SQLDB_HOST")
	username := os.Getenv("DIWISE_SQLDB_USER")
	dbName := os.Getenv("DIWISE_SQLDB_NAME")
	password := os.Getenv("DIWISE_SQLDB_PASSWORD")
	sslMode := getEnv("DIWISE_SQLDB_SSLMODE", "require")

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
		db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
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

	db.impl.AutoMigrate(&models.DeviceControlledProperty{})
	db.impl.AutoMigrate(&models.DeviceModel{})
	db.impl.AutoMigrate(&models.DeviceValue{})
	db.impl.AutoMigrate(&models.Device{})

	db.impl.Model(&models.DeviceModel{}).Association("DeviceControlledProperty")
	db.impl.Model(&models.Device{}).Association("DeviceModel")

	// Make sure that the controlled properties table is properly seeded
	props := map[string]string{
		"state":        "",
		"fillingLevel": "l",
		"snowDepth":    "snow",
		"temperature":  "t",
	}

	for property, abbreviation := range props {
		controlledProperty := models.DeviceControlledProperty{}

		result := db.impl.Where("name = ?", property).First(&controlledProperty)
		if result.RowsAffected == 0 {
			log.Infof("ControlledProperty %s not found in database. Creating ...", property)

			controlledProperty.Name = property
			controlledProperty.Abbreviation = abbreviation
			result = db.impl.Create(&controlledProperty)

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
		result := db.impl.Where("device_model_id = ?", model.DeviceModelID).First(&m)
		if result.RowsAffected == 0 {
			m.DeviceModelID = model.DeviceModelID
			m.Category = model.Category
			m.ControlledProperties = model.ControlledProperties

			result = db.impl.Create(&m)
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

	// TODO: Separate fiware.Device from the repository layer so that we do not
	// have to deal with ID strings like this
	if !strings.HasPrefix(src.ID, fiware.DeviceIDPrefix) {
		return nil, fmt.Errorf("device id %s must start with \"%s\"", src.ID, fiware.DeviceIDPrefix)
	}

	// Truncate the leading fiware prefix from the device id string
	shortDeviceID := src.ID[len(fiware.DeviceIDPrefix):]

	if src.RefDeviceModel == nil {
		return nil, fmt.Errorf("CreateDevice requires non-empty device model")
	}

	deviceModel, err := db.getDeviceModelFromString(src.RefDeviceModel.Object)
	if err != nil {
		return nil, err
	}

	device := &models.Device{
		DeviceID:    shortDeviceID,
		DeviceModel: *deviceModel,
	}

	if src.Location != nil {
		pt := src.Location.Value.GetAsPoint()
		device.Longitude = pt.Coordinates[0]
		device.Latitude = pt.Coordinates[1]
	}

	result := db.impl.Create(device)
	if result.Error != nil {
		return nil, result.Error
	}

	return device, nil
}

func (db *myDB) CreateDeviceModel(src *fiware.DeviceModel) (*models.DeviceModel, error) {

	// TODO: Separate fiware.DeviceModel from the repository layer so that we do not
	// have to deal with ID strings like this
	if !strings.HasPrefix(src.ID, fiware.DeviceModelIDPrefix) {
		return nil, fmt.Errorf("device id %s must start with \"%s\"", src.ID, fiware.DeviceModelIDPrefix)
	}

	// Truncate the leading fiware prefix from the device model id string
	shortDeviceID := src.ID[len(fiware.DeviceModelIDPrefix):]

	if src.ControlledProperty == nil {
		return nil, fmt.Errorf("creating device model is not allowed without controlled properties")
	}

	controlledProperties, err := db.getControlledProperties(src.ControlledProperty.Value)
	if err != nil {
		return nil, fmt.Errorf("controlled property is not supported: %s", err.Error())
	}

	if src.Category == nil {
		return nil, fmt.Errorf("creating device model is not allowed without a specified category")
	}

	deviceModel := &models.DeviceModel{
		DeviceModelID:        shortDeviceID,
		Category:             src.Category.Value[0],
		ControlledProperties: controlledProperties,
	}

	if src.BrandName != nil {
		deviceModel.BrandName = src.BrandName.Value
	}

	if src.ModelName != nil {
		deviceModel.ModelName = src.ModelName.Value
	}

	if src.ManufacturerName != nil {
		deviceModel.ManufacturerName = src.ManufacturerName.Value
	}

	if src.Name != nil {
		deviceModel.Name = src.Name.Value
	}

	result := db.impl.Create(deviceModel)
	if result.Error != nil {
		return nil, result.Error
	}

	return deviceModel, nil
}

func (db *myDB) GetDeviceFromID(id string) (*models.Device, error) {
	device := &models.Device{DeviceID: id}
	result := db.impl.Where(device).First(device)
	if result.Error != nil {
		return nil, result.Error
	}

	deviceValue := &models.DeviceValue{DeviceID: device.ID}
	deviceValues := []models.DeviceValue{}

	// TODO: DISTINCT ON is PostgreSQL specific and fails on SQLite. Find a compatible solution.
	result = db.impl.Select("DISTINCT ON (device_controlled_property_id) device_controlled_property_id, value").Where(deviceValue).Order("device_controlled_property_id, observed_at desc").Find(&deviceValues)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected > 0 {
		values := []string{}

		for _, value := range deviceValues {
			for _, controlledProperty := range db.controlledProperties {
				if controlledProperty.ID == value.DeviceControlledPropertyID {
					if len(controlledProperty.Abbreviation) > 0 {
						values = append(values, fmt.Sprintf("%s=%s", controlledProperty.Abbreviation, value.Value))
					} else {
						values = append(values, value.Value)
					}
				}
			}
		}

		device.Value = strings.Join(values, ";")
	}

	// TODO: Remove this temporary quick fix after the erroneous seed data is fixed
	if device.Longitude > device.Latitude {
		swap := device.Latitude
		device.Latitude = device.Longitude
		device.Longitude = swap
	}

	return device, nil
}

func (db *myDB) GetDevices() ([]models.Device, error) {
	devices := []models.Device{}
	result := db.impl.Order("device_id").Find(&devices)
	if result.Error != nil {
		return nil, result.Error
	}

	for idx, d := range devices {
		d, err := db.GetDeviceFromID(d.DeviceID)
		if err == nil {
			devices[idx] = *d
		}
	}

	return devices, nil
}

func (db *myDB) GetDeviceModels() ([]models.DeviceModel, error) {
	deviceModels := []models.DeviceModel{}
	result := db.impl.Order("device_model_id").Find(&deviceModels)
	if result.Error != nil {
		return nil, result.Error
	}
	return deviceModels, nil
}

func (db *myDB) GetDeviceModelFromID(id string) (*models.DeviceModel, error) {
	deviceModel := &models.DeviceModel{DeviceModelID: id}
	result := db.impl.Where(deviceModel).First(deviceModel)
	if result.Error != nil {
		return nil, result.Error
	}
	return deviceModel, nil
}

func (db *myDB) GetDeviceModelFromPrimaryKey(id uint) (*models.DeviceModel, error) {
	deviceModel := &models.DeviceModel{}
	result := db.impl.Find(deviceModel, id)
	if result.Error != nil {
		return nil, result.Error
	}

	return deviceModel, nil
}

func (db *myDB) UpdateDeviceValue(deviceID, value string) error {
	// Make sure that we have a corresponding device ...
	device := &models.Device{}
	result := db.impl.Where("device_id = ?", deviceID).First(device)
	if result.Error != nil {
		return result.Error
	} else if result.RowsAffected != 1 {
		//TODO: We need error groups and introduce a NOT FOUND error here
		return errors.New("attempt to update non existing device")
	}

	// Get the corresponding device model
	deviceModel := &models.DeviceModel{}
	result = db.impl.Preload("ControlledProperties").Find(deviceModel, device.DeviceModelID)
	if result.Error != nil {
		return result.Error
	} else if result.RowsAffected != 1 {
		return fmt.Errorf("failed to find corresponding device model for device %s", deviceID)
	}

	// Build a lookup table for controlled property abbrevations to db primary keys
	ctrlPropMap := map[string]uint{}
	for _, prop := range deviceModel.ControlledProperties {
		ctrlPropMap[prop.Abbreviation] = prop.ID
	}

	// TODO: Check that all values are supported before starting to add them
	// TODO: Support a delta to not store too small changes

	timeNow := time.Now().UTC()

	for _, v := range strings.Split(value, ";") {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			// If the value can not be split around an equal sign
			if isStateValue(v) {
				// ... and the value is a state value. Then we create a new tuple manually to
				// link the value to the "state" property
				kv = []string{"", v}
			} else {
				return fmt.Errorf("unable to store value %s. Failed to split value in two", v)
			}
		}

		if ctrlPropMap[kv[0]] == 0 {
			return fmt.Errorf("device %s does not support this controlled property: %s", deviceID, kv[0])
		}

		deviceValue := &models.DeviceValue{
			DeviceID:                   device.ID,
			DeviceControlledPropertyID: ctrlPropMap[kv[0]],
			Value:                      kv[1],
			ObservedAt:                 timeNow,
		}

		result = db.impl.Create(deviceValue)
		if result.Error != nil {
			return result.Error
		}
	}

	db.impl.Model(&models.Device{}).Where("id = ?", device.ID).Update("date_last_value_reported", timeNow)

	return nil
}

func isStateValue(value string) bool {
	return (strings.Compare(value, "on") == 0 || strings.Compare(value, "off") == 0)
}

func (db *myDB) getControlledProperties(properties []string) ([]models.DeviceControlledProperty, error) {
	found := []models.DeviceControlledProperty{}

	for _, p := range properties {
		for _, controlledProperty := range db.controlledProperties {
			if controlledProperty.Name == p {
				found = append(found, controlledProperty)
				break
			}
		}
	}

	if len(found) != len(properties) {
		return nil, fmt.Errorf("unable to find all controlled properties %v", properties)
	}

	return found, nil
}

func (db *myDB) getDeviceModelFromString(deviceModelID string) (*models.DeviceModel, error) {
	truncatedID := deviceModelID

	if strings.HasPrefix(deviceModelID, fiware.DeviceModelIDPrefix) {
		truncatedID = deviceModelID[len(fiware.DeviceModelIDPrefix):]
	}

	m := &models.DeviceModel{}
	result := db.impl.Where("device_model_id = ?", truncatedID).First(m)
	if result.RowsAffected == 1 {
		return m, nil
	}

	return nil, errors.New("No DeviceModel found matching " + deviceModelID)
}
