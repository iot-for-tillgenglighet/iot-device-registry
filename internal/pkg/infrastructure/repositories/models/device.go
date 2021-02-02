package models

import (
	"gorm.io/gorm"
)

type Device struct {
	gorm.Model
	DeviceID      string `gorm:"unique"`
	Latitude      float64
	Longitude     float64
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

//DeviceControlledProperty stores different properties that devices can control/sense/meter/whatever
type DeviceControlledProperty struct {
	gorm.Model
	Name         string `gorm:"unique"`
	Abbreviation string
}
