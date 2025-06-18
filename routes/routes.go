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
	api.Get("/register-masjid/:id_event", controllers.GetMasjidList)
	api.Get("/rekap-absen/:id_masjid", controllers.GetRekapAbsen)
	api.Get("/rekap-absen-sholat/:id_masjid", controllers.GetRekapSholat)
	api.Get("/get-masjid/:id_masjid", controllers.GetMasjidByID)
	api.Get("/statistics-event", controllers.GetEventStatistics)
	api.Get("/dashboard", controllers.GetNewRegistrantStatistics)
	api.Get("/statistics-event-all", controllers.GetAttendanceStatistics)
	api.Get("/statistics-rekap-masjid", controllers.GetRekapPerMasjid)
	api.Get("/statistics-absensi", controllers.GetAbsensiStatistics)

	apiV1 := app.Group("/api/v1")
	apiV1.Post("/absent-qr", ApiKeyMiddleware, controllers.SaveAbsenQR)
	apiV1.Post("/collections-create", controllers.CreateCollection)
	apiV1.Get("/collections-get-absensi/:slug", controllers.ViewCollection)
	apiV1.Get("/collections-get", controllers.GetCollectionsMeta)
	apiV1.Get("/collections-get-meta/:slug", controllers.GetCollectionsMetaDetail)
	apiV1.Post("/collections/add-peserta", controllers.AddPesertaToCollection)
	apiV1.Get("/data-peserta-masjid", controllers.GetPesertaDanMasjid)

	user := api.Group("/users", middlewares.JWTMiddleware())
	user.Get("/profile", controllers.GetProfile)
}
