package models

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type Device struct {
	gorm.Model
	DeviceID  string
	Latitude  float64
	Longitude float64
}
