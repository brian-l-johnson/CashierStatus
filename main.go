package main

import (
	"fmt"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/brian-l-johnson/CashierStatusBoard/v2/server"
)

func main() {
	fmt.Println("starting up")
	models.Init()
	server.Init()
}
