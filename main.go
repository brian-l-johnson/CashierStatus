package main

import (
	"fmt"
	"log"
	"os"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/brian-l-johnson/CashierStatusBoard/v2/server"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Failed to load .env")
	}
	fmt.Printf("allowed web socket orgins: %v\n", os.Getenv("WEBSOCKET_ALLOWED_ORIGINS"))

	fmt.Println("starting up")
	models.Init()
	server.Init()
}
