package models

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

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
	db.AutoMigrate(&User{})

	var user User
	result := db.First(&user, "name=?", "admin")
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		fmt.Println("Admin user does not exist, creating...")

		adminUser := MakeUser("admin")

		genpw, err := GenerateRandomString(32)
		if err != nil {
			panic("unable to generate random password")
		}
		adminUser.SetPassword(genpw)
		adminUser.Active = true
		adminUser.Roles = append(user.Roles, "admin", "view", "update")

		result = db.Create(&adminUser)
		if result.Error != nil {
			panic("unable to save admin user")
		}
		fmt.Printf("created 'admin' user with a password of '%v'\n", genpw)
	}
}

func GetDB() *gorm.DB {
	return db
}

func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret), nil
}
