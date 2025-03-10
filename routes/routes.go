package routes

import (
	"shollu/controllers"
	"shollu/middlewares"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Post("/login", controllers.Login)

	user := api.Group("/users", middlewares.JWTMiddleware())
	user.Get("/profile", controllers.GetProfile)
}
