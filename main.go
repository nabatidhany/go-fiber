package main

import (
	"shollu/config"
	"shollu/database"
	"shollu/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	config.LoadConfig()
	database.Connect()

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Izinkan semua domain (ganti dengan domain frontend jika perlu)
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Options("*", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent) // 204 No Content untuk preflight
	})

	routes.SetupRoutes(app)

	app.Listen("0.0.0.0:3000")
}
