package models

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func Init() {
	fmt.Println("initing db...")
	var err error
	db, err = gorm.Open(sqlite.Open("cashierstatus.db"), &gorm.Config{})
	if err != nil {
		panic("failed to open database file")
	}
	db.AutoMigrate(&Cashier{})
}

func GetDB() *gorm.DB {
	return db
}
