package models

import (
	"github.com/jinzhu/gorm"
)

type Device struct {
	gorm.Model
	DeviceID      string
	DeviceModelID string
	Latitude      float64
	Longitude     float64
}
