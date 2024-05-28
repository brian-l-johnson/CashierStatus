package models

import "gorm.io/gorm"

type Cashier struct {
	gorm.Model `json:"-"`
	ID         uint
	Serving    string
}
