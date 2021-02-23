package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/iot-device-registry/internal/pkg/infrastructure/repositories/models"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
	"github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/datamodels/fiware"
	ngsi "github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/ngsi-ld"
	ngsitypes "github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/ngsi-ld/types"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestThatCreateEntityDoesNotAcceptUnknownBody(t *testing.T) {
	bodyContents := []byte("{\"json\":\"json\"}")
	req, _ := http.NewRequest("POST", createURL("/ngsi-ld/v1/entities"), bytes.NewBuffer(bodyContents))
	w := httptest.NewRecorder()
	log := logging.NewLogger()

	ctxreg := createContextRegistry(log, nil, nil)
	ngsi.NewCreateEntityHandler(ctxreg).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Error("CreateEntity did not return a BadRequest status.")
	}
}

func TestThatCreateEntityStoresCorrectDevice(t *testing.T) {
	db := &dbMock{}
	deviceID := fiware.DeviceIDPrefix + "deviceID"
	device := fiware.NewDevice(deviceID, "")
	device.RefDeviceModel, _ = fiware.NewDeviceModelRelationship(
		fiware.DeviceModelIDPrefix + "livboj",
	)
	jsonBytes, _ := json.Marshal(device)
	log := logging.NewLogger()

	req, _ := http.NewRequest("POST", createURL("/ngsi-ld/v1/entities"), bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	ctxreg := createContextRegistry(log, nil, db)
	ngsi.NewCreateEntityHandler(ctxreg).ServeHTTP(w, req)

	if db.createCount != 1 {
		t.Error("CreateCount should be 1, but was ", db.createCount, "!")
	}

	if db.device.ID != deviceID {
		t.Error("DeviceID should be " + deviceID + ", but was " + db.device.ID)
	}
}

func TestThatCreateEntityStoresCorrectDeviceModel(t *testing.T) {
	db := &dbMock{}

	categories := []string{"sensor"}
	deviceModel := fiware.NewDeviceModel("badtemperatur", categories)
	deviceModel.ControlledProperty = ngsitypes.NewTextListProperty([]string{"temperature"})

	jsonBytes, _ := json.Marshal(deviceModel)
	log := logging.NewLogger()

	req, _ := http.NewRequest("POST", createURL("/ngsi-ld/v1/entities"), bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	ctxreg := createContextRegistry(log, nil, db)
	ngsi.NewCreateEntityHandler(ctxreg).ServeHTTP(w, req)

	if db.createCount != 1 {
		t.Error("CreateCount should be 1, but was ", db.createCount, "!")
	}
}

func TestThatCreateEntityFailsOnUnknownEntity(t *testing.T) {
	db := &dbMock{
		createDeviceModelError: errors.New("test"),
	}

	categories := []string{"sensor"}
	deviceModel := fiware.NewDeviceModel("badtemperatur", categories)
	deviceModel.ControlledProperty = ngsitypes.NewTextListProperty([]string{"temperature"})

	jsonBytes, _ := json.Marshal(deviceModel)
	log := logging.NewLogger()

	req, _ := http.NewRequest("POST", createURL("/ngsi-ld/v1/entities"), bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()

	ctxreg := createContextRegistry(log, nil, db)
	ngsi.NewCreateEntityHandler(ctxreg).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Error("CreateEntity did not return a BadRequest status.")
	}
}

func TestThatPatchWaterTempDevicePublishesOnTheMessageQueue(t *testing.T) {
	db := &dbMock{
		deviceFromID: &models.Device{},
	}
	m := msgMock{}

	jsonBytes, _ := json.Marshal(createDevicePatchWithValue("sk-elt-temp-02", "t%3D12"))
	req, _ := http.NewRequest("PATCH", createURL("/ngsi-ld/v1/entities/urn:ngsi-ld:Device:sk-elt-temp-02/attrs/"), bytes.NewBuffer(jsonBytes))
	w := httptest.NewRecorder()
	log := logging.NewLogger()

	ctxreg := createContextRegistry(log, &m, db)
	ngsi.NewUpdateEntityAttributesHandler(ctxreg).ServeHTTP(w, req)

	if m.PublishCount != 1 {
		t.Error("Wrong publish count: ", m.PublishCount, "!=", 1)
	}
}

func TestRetrieveEntity(t *testing.T) {
	db := &dbMock{
		deviceFromID:        &models.Device{},
		deviceModelReturned: &models.DeviceModel{},
	}

	log := logging.NewLogger()
	req, _ := http.NewRequest("GET", createURL("/ngsi-ld/v1/entities/urn:ngsi-ld:DeviceModel:sk-elt-temp-02"), nil)
	w := httptest.NewRecorder()

	ctxreg := createContextRegistry(log, nil, db)

	ngsi.NewRetrieveEntityHandler(ctxreg).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Request failed: %d", w.Code)
	}
}

// write unit test for retrieve entity where device is nil.

func createDevicePatchWithValue(deviceid, value string) *fiware.Device {
	device := fiware.NewDevice(deviceid, value)
	return device
}

func createURL(path string, params ...string) string {
	url := "http://localhost:8080/ngsi-ld/v1" + path

	if len(params) > 0 {
		url = url + "?"

		for _, p := range params {
			url = url + p + "&"
		}

		url = strings.TrimSuffix(url, "&")
	}

	return url
}

type msgMock struct {
	PublishCount uint32
}

func (m *msgMock) PublishOnTopic(message messaging.TopicMessage) error {
	m.PublishCount++
	return nil
}

type dbMock struct {
	createCount              uint32
	device                   *fiware.Device
	deviceModel              *fiware.DeviceModel
	createDeviceModelError   error
	deviceFromID             *models.Device
	deviceFromIDError        error
	deviceModelReturned      *models.DeviceModel
	deviceModelReturnedError error
}

func (db *dbMock) CreateDevice(device *fiware.Device) (*models.Device, error) {
	db.createCount++
	db.device = device

	return nil, nil
}

func (db *dbMock) CreateDeviceModel(deviceModel *fiware.DeviceModel) (*models.DeviceModel, error) {
	if db.createDeviceModelError != nil {
		return nil, db.createDeviceModelError
	}

	db.createCount++
	db.deviceModel = deviceModel

	return nil, nil
}

func (db *dbMock) GetDeviceFromID(id string) (*models.Device, error) {
	if db.deviceFromID != nil || db.deviceFromIDError != nil {
		return db.deviceFromID, db.deviceFromIDError
	}

	return nil, fmt.Errorf("Unexpected call to GetDeviceFromID with id %s", id)
}

func (db *dbMock) GetDevices() ([]models.Device, error) {
	return []models.Device{}, nil
}

func (db *dbMock) GetDeviceModels() ([]models.DeviceModel, error) {
	return []models.DeviceModel{}, nil
}

func (db *dbMock) GetDeviceModelFromID(id string) (*models.DeviceModel, error) {
	return db.deviceModelReturned, db.deviceModelReturnedError
}

func (db *dbMock) GetDeviceModelFromPrimaryKey(id uint) (*models.DeviceModel, error) {
	return db.deviceModelReturned, db.deviceModelReturnedError
}

func (db *dbMock) UpdateDeviceValue(deviceID, value string) error {
	return nil
}
