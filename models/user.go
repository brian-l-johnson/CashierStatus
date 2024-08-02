package models

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Roles []string

type User struct {
	gorm.Model   `json:"-"`
	Name         string
	PasswordHash string `json:"-"`
	Active       bool
	Roles        Roles `gorm:"type:VARCHAR(255)"`
	UID          string
}

func MakeUser(name string) User {
	var user User
	user.Name = name
	user.Active = false
	user.UID = uuid.New().String()
	user.Roles = append(user.Roles, "viewer")

	return user
}

func (u *User) SetPassword(pw string) {
	fmt.Printf("setting password to: %v\n", pw)
	bytes, hasherr := bcrypt.GenerateFromPassword([]byte(pw), 14)
	if hasherr != nil {
		panic("unable to hash password")
	}
	u.PasswordHash = string(bytes)
}

func (u *User) CheckPassword(pw string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pw))
	return err == nil
}

func (r *Roles) Scan(src any) error {
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	case nil:
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}

	*r = strings.Split(string(data), ",")

	return nil
}

func (r Roles) Value() (driver.Value, error) {
	if len(r) == 0 {
		return nil, nil
	}
	return strings.Join(r, ","), nil
}
