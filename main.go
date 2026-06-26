package main

import (
	"fmt"
	"log"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/brian-l-johnson/CashierStatusBoard/v2/server"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Failed to load .env")
	}

	fmt.Println("starting up")
	models.Init()
	server.Init()
}
