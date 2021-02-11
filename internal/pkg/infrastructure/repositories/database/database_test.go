package database

import (
	"fmt"
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
		device := fiware.NewDevice("ID1", "Value")

		_, err := db.CreateDevice(device)
		if err == nil || strings.Compare(err.Error(), "CreateDevice requires non-empty device model") != 0 {
			t.Error(err.Error())
		}
	}
}

func TestThatCreateDeviceFailsWithUnknownDeviceModel(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		device := fiware.NewDevice("ID2", "Value")
		var err error

		device.RefDeviceModel, err = types.NewDeviceModelRelationship("urn:ngsi-ld:DeviceModel:refDeviceModel")

		_, err = db.CreateDevice(device)
		if err == nil || strings.Compare(err.Error(), "No DeviceModel found matching urn:ngsi-ld:DeviceModel:refDeviceModel") != 0 {
			t.Error(err.Error())
		}
	}
}

func TestThatCreateDeviceModelStoresAllValues(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		brandName := "galaxy"
		modelName := "S20"
		manufacturerName := "samsung"
		name := "ourModel"

		categories := []string{"temperature"}
		deviceModel := fiware.NewDeviceModel("ID3", categories)
		deviceModel.ControlledProperty = types.NewTextListProperty([]string{"temperature"})
		deviceModel.BrandName = types.NewTextProperty(brandName)
		deviceModel.ModelName = types.NewTextProperty(modelName)
		deviceModel.ManufacturerName = types.NewTextProperty(manufacturerName)
		deviceModel.Name = types.NewTextProperty(name)

		createdDeviceModel, err := db.CreateDeviceModel(deviceModel)
		if err != nil {
			t.Error("CreateDeviceModel test failed:" + err.Error())
		}

		// get deviceModel and compare
		createdDeviceModel, err = db.GetDeviceModelFromID(createdDeviceModel.ID)
		if err != nil {
			t.Error("GetDeviceModelFromID failed:" + err.Error())
		}

		checkStringValue(t, "brand name", createdDeviceModel.BrandName, brandName)
		checkStringValue(t, "model name", createdDeviceModel.ModelName, modelName)
		checkStringValue(t, "manufacturer name", createdDeviceModel.ManufacturerName, manufacturerName)
		checkStringValue(t, "name", createdDeviceModel.Name, name)
	}
}

func checkStringValue(t *testing.T, property, lhs, rhs string) {
	if strings.Compare(lhs, rhs) != 0 {
		t.Errorf("Check string failed for property %s: %s != %s", property, lhs, rhs)
	}
}

func TestGetDeviceModels(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		_, err := db.GetDeviceModels()
		if err != nil {
			t.Error("Failed to get DeviceModels")
		}
	}
}

func TestGetDeviceModelFromID(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		categories := []string{"temperature"}
		deviceModel := fiware.NewDeviceModel("ID4", categories)
		deviceModel.ControlledProperty = types.NewTextListProperty([]string{"temperature"})

		dM, _ := db.CreateDeviceModel(deviceModel)

		dM2, err := db.GetDeviceModelFromID(dM.ID)
		if err != nil {
			t.Error("GetDeviceModelFromID failed with error:", err.Error())
		}

		if strings.Compare(dM.DeviceModelID, dM2.DeviceModelID) != 0 {
			t.Error(fmt.Sprintf("DeviceModelFromID returned incorrect DeviceModel \"%s\" != \"%s\"", dM.DeviceModelID, dM2.DeviceModelID))
		}
	}
}

func TestCreateDevice(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		categories := []string{"T"}
		deviceModel := fiware.NewDeviceModel("ID5", categories)
		deviceModel.ControlledProperty = types.NewTextListProperty([]string{"temperature"})

		_, err := db.CreateDeviceModel(deviceModel)
		if err != nil {
			t.Error("CreateDevice test failed to create device model:" + err.Error())
		}

		device := fiware.NewDevice("ID6", "Value")

		device.RefDeviceModel, err = types.NewDeviceModelRelationship(deviceModel.ID)
		_, err = db.CreateDevice(device)
		if err != nil {
			t.Error("CreateDevice test failed:" + err.Error())
		}
	}
}

func TestCreateDeviceModelForWaterTemperatureDevice(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {

		categories := []string{"sensor"}
		deviceModel := fiware.NewDeviceModel("badtemperatur", categories)
		deviceModel.ControlledProperty = types.NewTextListProperty([]string{"temperature"})

		_, err := db.CreateDeviceModel(deviceModel)
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
}

func TestThatCreateDeviceModelFailsOnUnknownControlledProperty(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		categories := []string{"sensor"}
		deviceModel := fiware.NewDeviceModel("badtemperatur", categories)
		deviceModel.ControlledProperty = types.NewTextListProperty([]string{"spaceship"})

		_, err := db.CreateDeviceModel(deviceModel)
		if err == nil || strings.Compare(err.Error(), "Controlled property is not supported: Unable to find all controlled properties [spaceship]") != 0 {
			t.Error("CreateDeviceModelUnknownControlledProperty test failed:" + err.Error())
		}
	}
}

func TestGetDevices(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		categories := []string{"T"}
		deviceModel := fiware.NewDeviceModel("ID7", categories)
		deviceModel.ControlledProperty = types.NewTextListProperty([]string{"temperature"})

		_, err := db.CreateDeviceModel(deviceModel)
		if err != nil {
			t.Error("CreateDevice test failed to create device model:" + err.Error())
		}

		device := fiware.NewDevice("ID8", "Value")

		device.RefDeviceModel, err = types.NewDeviceModelRelationship(deviceModel.ID)
		_, err = db.CreateDevice(device)

		_, err = db.GetDevices()
		if err != nil {
			t.Error("Failed")
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
