package routes

import (
	"shollu/controllers"
	"shollu/middlewares"

	"github.com/gofiber/fiber/v2"
)

func ApiKeyMiddleware(c *fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")          // Ambil API Key dari header
	validApiKey := "shollusemakindidepan" // Ganti dengan API Key yang aman

	if apiKey != validApiKey {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: Invalid API Key"})
	}
	return c.Next()
}

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Post("/login", controllers.Login)
	api.Post("/register", controllers.Register)
	api.Post("/register-itikaf", controllers.RegisterPesertaItikaf)

	apiV1 := app.Group("/api/v1")
	apiV1.Post("/absent-qr", ApiKeyMiddleware, controllers.SaveAbsenQR)

	user := api.Group("/users", middlewares.JWTMiddleware())
	user.Get("/profile", controllers.GetProfile)
}
