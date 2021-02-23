package models

import (
	"time"

	"gorm.io/gorm"
)

//Device is the database model to store devices in our database
type Device struct {
	gorm.Model
	DeviceID      string `gorm:"unique"`
	Latitude      float64
	Longitude     float64
	Value         string
	DeviceModelID uint
	DeviceModel   DeviceModel
}

//DeviceModel is the database model to store Fiware Device Models in our database
type DeviceModel struct {
	gorm.Model
	DeviceModelID        string `gorm:"unique"`
	BrandName            string
	ModelName            string
	ManufacturerName     string
	Name                 string
	Category             string
	ControlledProperties []DeviceControlledProperty `gorm:"many2many:devicemodel_ctrlprops;"`
}

//DeviceValue stores the value from a point in time (observedAt)
type DeviceValue struct {
	gorm.Model
	DeviceID                   uint `gorm:"index:values_from_device"`
	DeviceControlledPropertyID uint `gorm:"index:values_from_property"`
	Value                      string
	ObservedAt                 time.Time
}

//DeviceControlledProperty stores different properties that devices can control/sense/meter/whatever
type DeviceControlledProperty struct {
	gorm.Model
	Name         string `gorm:"unique"`
	Abbreviation string
}
