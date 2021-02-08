package database

import (
	"os"
	"strings"
	"testing"

	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/ngsi-ld/types"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestThatCreateDeviceReturnsErrorIfDeviceModelIsNil(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		device := fiware.NewDevice("ID", "Value")

		_, err := db.CreateDevice(device)
		if err == nil || strings.Compare(err.Error(), "CreateDevice requires non-empty device model") != 0 {
			t.Error(err.Error())
		}
	}
}

func TestThatCreateDeviceFailsWithUnknownDeviceModel(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		device := fiware.NewDevice("ID", "Value")
		var err error

		device.RefDeviceModel, err = types.NewDeviceModelRelationship("urn:ngsi-ld:DeviceModel:refDeviceModel")

		_, err = db.CreateDevice(device)
		if err == nil || strings.Compare(err.Error(), "No DeviceModel found matching urn:ngsi-ld:DeviceModel:refDeviceModel") != 0 {
			t.Error(err.Error())
		}
	}
}

func TestCreateDeviceModel(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		categories := []string{"temperature"}
		deviceModel := fiware.NewDeviceModel("ID", categories)
		deviceModel.ControlledProperty = types.NewTextListProperty([]string{"temperature"})

		_, err := db.CreateDeviceModel(deviceModel)
		if err != nil {
			t.Error("CreateDeviceModel test failed:" + err.Error())
		}
	}
}

func TestCreateDevice(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		categories := []string{"T"}
		deviceModel := fiware.NewDeviceModel("ID2", categories)

		_, err := db.CreateDeviceModel(deviceModel)
		if err != nil {
			t.Error("CreateDevice test failed to create device model:" + err.Error())
		}

		device := fiware.NewDevice("ID", "Value")

		device.RefDeviceModel, err = types.NewDeviceModelRelationship(deviceModel.ID)
		_, err = db.CreateDevice(device)
		if err != nil {
			t.Error("CreateDevice test failed:" + err.Error())
		}
	}
}

func TestCreateDeviceModelForWaterTemperatureDevice(t *testing.T) {
	db, err := NewDatabaseConnection(NewSQLiteConnector())

	if err != nil {
		t.Error(err.Error())
	}

	categories := []string{"sensor"}
	deviceModel := fiware.NewDeviceModel("badtemperatur", categories)
	deviceModel.ControlledProperty = types.NewTextListProperty([]string{"temperature"})

	_, err = db.CreateDeviceModel(deviceModel)
	if err != nil {
		t.Error("CreateDevice test failed:" + err.Error())
	}

	device := fiware.NewDevice("badtemperatur", "18.5")

	device.RefDeviceModel, err = types.NewDeviceModelRelationship(deviceModel.ID)
	_, err = db.CreateDevice(device)
	if err != nil {
		t.Error("CreateDevice test failed:" + err.Error())
	}
}

func TestThatCreateDeviceModelFailsOnUnknownControlledProperty(t *testing.T) {
	db, err := NewDatabaseConnection(NewSQLiteConnector())

	if err != nil {
		t.Error(err.Error())
	}

	categories := []string{"sensor"}
	deviceModel := fiware.NewDeviceModel("badtemperatur", categories)
	deviceModel.ControlledProperty = types.NewTextListProperty([]string{"spaceship"})

	_, err = db.CreateDeviceModel(deviceModel)
	if err == nil || strings.Compare(err.Error(), "Controlled property is not supported: Unable to find all controlled properties [spaceship]") != 0 {
		t.Error("CreateDeviceModelUnknownControlledProperty test failed:" + err.Error())
	}
}

func newDatabaseForTest(t *testing.T) (Datastore, bool) {
	log := logging.NewLogger()
	db, err := NewDatabaseConnection(NewSQLiteConnector(), log)

	if err != nil {
		t.Error(err.Error())
		return nil, false
	}

	return db, true
}
