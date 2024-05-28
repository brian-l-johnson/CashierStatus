package models

import (
	"time"
)

type OrderReq struct {
	OrderNum  string    `json:"ordernum"`
	OrderTime time.Time `json:"ordertime"`
}
