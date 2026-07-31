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
	// WAL instead of the default rollback journal, so readers do not block on
	// the in-flight writer. Steady-state read load is light -- cashier boards
	// get updates pushed over SSE and only read /cashiers on (re)connect, and
	// info boards poll /notes once a minute. The case this actually buys us is
	// the burst: when the server restarts or the network blips, every board
	// reconnects and re-reads /cashiers at the same moment, which is also when
	// cashier updates are most likely to be writing. Under a rollback journal
	// those reads serialize behind the writer. Headroom, not a fix for
	// anything observed. The driver already sets busy_timeout(5000) itself.
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
