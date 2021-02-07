package database

import (
	"encoding/json"
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

func newDatabaseForTest(t *testing.T) (Datastore, bool) {
	log := logging.NewLogger()
	db, err := NewDatabaseConnection(NewSQLiteConnector(), log)

	if err != nil {
		t.Error(err.Error())
		return nil, false
	}

	return db, true
}

func deviceFromJSON(deviceJSON string) (*fiware.Device, error) {
	strToByte := []byte(deviceJSON)
	device := &fiware.Device{}
	err := json.Unmarshal(strToByte, device)

	return device, err
}
