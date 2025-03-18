package main

import (
	"regexp"
	"shollu/config"
	"shollu/database"
	"shollu/routes"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Regex untuk mengganti angka dalam path menjadi ":id"
var dynamicPathPattern = regexp.MustCompile(`/\d+`)

// Fungsi untuk normalisasi path
func normalizePath(path string) string {
	return dynamicPathPattern.ReplaceAllString(path, "/:id")
}

// Buat metrik Prometheus
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total jumlah HTTP request",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Durasi HTTP request dalam detik",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	// Register metrik ke Prometheus
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

func main() {
	config.LoadConfig()
	database.Connect()

	app := fiber.New()

	// Middleware untuk mengukur request dan latensi
	app.Use(func(c *fiber.Ctx) error {
		method := c.Method()
		path := normalizePath(c.Path()) // Normalisasi path

		histogram := httpRequestDuration.WithLabelValues(method, path)
		timer := prometheus.NewTimer(histogram)
		defer timer.ObserveDuration()

		err := c.Next()

		status := strconv.Itoa(c.Response().StatusCode())
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()

		return err
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Izinkan semua domain (ganti dengan domain frontend jika perlu)
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Options("*", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent) // 204 No Content untuk preflight
	})

	routes.SetupRoutes(app)

	// Endpoint `/metrics` untuk Prometheus
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	app.Listen("0.0.0.0:3000")
}
