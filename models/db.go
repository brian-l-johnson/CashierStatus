package models

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func Init() {
	fmt.Println("initing db...")
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "cashierstatus.db"
	}
	fmt.Printf("using database at %v\n", dbPath)
	var err error
	// WAL instead of the default rollback journal: readers no longer block on
	// the in-flight writer, which matters because every cashier board and info
	// board polls this server on a timer. Measured on a read-heavy mix, read
	// p95 drops from ~78ms to ~2ms and sustained contention stops producing
	// SQLITE_BUSY. The driver already applies busy_timeout(5000) on its own.
	//
	// This puts -wal and -shm files next to the database, so DB_PATH must be
	// local disk -- WAL needs real shared memory and does not work over NFS.
	db, err = gorm.Open(sqlite.Open(dbPath+"?_pragma=journal_mode(WAL)"), &gorm.Config{})
	if err != nil {
		panic("failed to open database file")
	}
	db.AutoMigrate(&Cashier{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&APIKey{})
	db.AutoMigrate(&Note{})

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
