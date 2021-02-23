package database

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/ngsi-ld/types"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestThatCreateDeviceReturnsErrorIfDeviceModelIsNil(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		device := newDevice()

		_, err := db.CreateDevice(device)

		errMsg := getErrorMessageOrString(err, "nil")
		if strings.Compare(errMsg, "CreateDevice requires non-empty device model") != 0 {
			t.Errorf("Unexpected error: %s", errMsg)
		}
	}
}

func TestThatCreateDeviceFailsWithUnknownDeviceModel(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		device := newDevice()
		device.RefDeviceModel, _ = fiware.NewDeviceModelRelationship(
			fiware.DeviceModelIDPrefix + "nosuchthing",
		)

		_, err := db.CreateDevice(device)

		errMsg := getErrorMessageOrString(err, "nil")
		if strings.Compare(errMsg, "No DeviceModel found matching urn:ngsi-ld:DeviceModel:nosuchthing") != 0 {
			t.Errorf("Unexpected error: %s", errMsg)
		}
	}
}

func TestThatCreateDeviceModelStoresAllValues(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		brandName := "galaxy"
		modelName := "S20"
		manufacturerName := "samsung"
		name := "ourModel"

		deviceModel := newDeviceModel()

		deviceModel.BrandName = types.NewTextProperty(brandName)
		deviceModel.ModelName = types.NewTextProperty(modelName)
		deviceModel.ManufacturerName = types.NewTextProperty(manufacturerName)
		deviceModel.Name = types.NewTextProperty(name)

		createdDeviceModel, err := db.CreateDeviceModel(deviceModel)
		if err != nil {
			t.Error("CreateDeviceModel test failed:" + err.Error())
		}

		// get deviceModel and compare
		createdDeviceModel, err = db.GetDeviceModelFromPrimaryKey(createdDeviceModel.ID)
		if err != nil {
			t.Error("GetDeviceModelFromPrimaryKey failed:" + err.Error())
		}

		checkStringValue(t, "brand name", createdDeviceModel.BrandName, brandName)
		checkStringValue(t, "model name", createdDeviceModel.ModelName, modelName)
		checkStringValue(t, "manufacturer name", createdDeviceModel.ManufacturerName, manufacturerName)
		checkStringValue(t, "name", createdDeviceModel.Name, name)
	}
}

func TestGetDeviceModels(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		if _, _, ok := seedNewDeviceModel(t, db); ok {

			models, _ := db.GetDeviceModels()

			if len(models) != 1 {
				t.Errorf("Returned number (%d) is different from expected %d.", len(models), 1)
			}
		}
	}
}

func TestGetDeviceModelFromPrimaryKey(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		if key, modelID, ok := seedNewDeviceModel(t, db); ok {

			deviceModel, err := db.GetDeviceModelFromPrimaryKey(key)
			if err != nil {
				t.Error("GetDeviceModelFromPrimaryKey failed with error:", err.Error())
			}

			if strings.Compare(modelID, deviceModel.DeviceModelID) != 0 {
				t.Error(fmt.Sprintf("DeviceModelFromID returned incorrect DeviceModel \"%s\" != \"%s\"",
					modelID, deviceModel.DeviceModelID,
				))
			}
		}
	}
}

func TestCreateDevice(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		if _, modelID, ok := seedNewDeviceModel(t, db); ok {

			var err error
			device := newDevice()
			device.RefDeviceModel, err = fiware.NewDeviceModelRelationship(
				fiware.DeviceModelIDPrefix + modelID,
			)

			_, err = db.CreateDevice(device)

			if err != nil {
				t.Error("CreateDevice test failed:" + err.Error())
			}
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
		device.RefDeviceModel, err = fiware.NewDeviceModelRelationship(deviceModel.ID)

		_, err = db.CreateDevice(device)

		if err != nil {
			t.Error("CreateDevice test failed:" + err.Error())
		}
	}
}

func TestThatCreateDeviceModelFailsOnUnknownControlledProperty(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		deviceModel := newDeviceModel()
		deviceModel.ControlledProperty = types.NewTextListProperty([]string{"spaceship"})

		_, err := db.CreateDeviceModel(deviceModel)

		errMsg := getErrorMessageOrString(err, "nil")
		if strings.Compare(errMsg, "Controlled property is not supported: Unable to find all controlled properties [spaceship]") != 0 {
			t.Error("CreateDeviceModelUnknownControlledProperty test failed:" + errMsg)
		}
	}
}

func TestGetDevices(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		if _, modelID, ok := seedNewDeviceModel(t, db); ok {
			device := newDevice()
			device.RefDeviceModel, _ = fiware.NewDeviceModelRelationship(
				fiware.DeviceModelIDPrefix + modelID,
			)
			db.CreateDevice(device)

			devices, _ := db.GetDevices()

			if len(devices) != 1 {
				t.Errorf("Number of returned devices (%d) does not match expected %d.", len(devices), 1)
			}
		}
	}
}

func TestUpdateDeviceValue(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		if _, deviceID, ok := seedNewDevice(t, db); ok {

			_ = db.UpdateDeviceValue(deviceID, "t=10")
			time.Sleep(10 * time.Millisecond)
			_ = db.UpdateDeviceValue(deviceID, "l=3")
			time.Sleep(10 * time.Millisecond)
			_ = db.UpdateDeviceValue(deviceID, "t=11")
			time.Sleep(10 * time.Millisecond)
			_ = db.UpdateDeviceValue(deviceID, "l=5")
			time.Sleep(10 * time.Millisecond)
			err := db.UpdateDeviceValue(deviceID, "t=12")

			if err != nil {
				t.Errorf("Failed to update device value: %s", err.Error())
			}

			// TODO: Fix SQLite incompatibility in GetDeviceFromID
			/*device, err = db.GetDeviceFromID(deviceID)
			if err != nil {
				t.Errorf("Failed to get device: %s", err.Error())
				return
			}
			if device.Value != "l=5;t=12" {
				t.Errorf("Received unexpected device value: %s", device.Value)
			}*/
		}
	}
}

func TestThatUpdateDeviceDoesNotSaveUnsupportedControlledProperty(t *testing.T) {
	if db, ok := newDatabaseForTest(t); ok {
		if _, deviceID, ok := seedNewDevice(t, db); ok {

			err := db.UpdateDeviceValue(deviceID, "snow=12")

			if err == nil {
				t.Error("Expected UpdateDeviceValue to fail, but it didn't.")
			}
		}
	}
}

func checkStringValue(t *testing.T, property, lhs, rhs string) {
	if strings.Compare(lhs, rhs) != 0 {
		t.Errorf("Check string failed for property %s: %s != %s", property, lhs, rhs)
	}
}

func getErrorMessageOrString(err error, orString string) string {
	if err != nil {
		return err.Error()
	}

	return orString
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

var numCreatedDevices int = 0

func newDevice() *fiware.Device {
	id := fmt.Sprintf("ID%d", numCreatedDevices)
	numCreatedDevices++

	return fiware.NewDevice(id, "on")
}

var numCreatedDeviceModels int = 0

func newDeviceModel() *fiware.DeviceModel {
	id := fmt.Sprintf("ID%d", numCreatedDeviceModels)
	numCreatedDeviceModels++

	categories := []string{"T"}
	deviceModel := fiware.NewDeviceModel(id, categories)
	deviceModel.ControlledProperty = types.NewTextListProperty([]string{"fillingLevel", "temperature"})

	return deviceModel
}

func seedNewDevice(t *testing.T, db Datastore) (uint, string, bool) {
	if _, modelID, ok := seedNewDeviceModel(t, db); ok {
		d := newDevice()
		d.RefDeviceModel, _ = fiware.NewDeviceModelRelationship(
			fiware.DeviceModelIDPrefix + modelID,
		)
		device, err := db.CreateDevice(d)

		if err != nil {
			t.Errorf("Failed to seed new device in database: %s", err.Error())
			return 0, "", false
		}

		return device.ID, device.DeviceID, true
	}

	return 0, "", false
}

func seedNewDeviceModel(t *testing.T, db Datastore) (uint, string, bool) {
	deviceModel, err := db.CreateDeviceModel(newDeviceModel())

	if err != nil {
		t.Errorf("Failed to seed device model in database: %s", err.Error())
		return 0, "", false
	}

	return deviceModel.ID, deviceModel.DeviceModelID, true
}
