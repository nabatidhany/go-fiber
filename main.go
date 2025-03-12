package main

import (
	"shollu/config"
	"shollu/database"
	"shollu/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	config.LoadConfig()
	database.Connect()

	app := fiber.New()
	routes.SetupRoutes(app)

	app.Listen("0.0.0.0:3000")
	// app.Listen(":8080")
}
