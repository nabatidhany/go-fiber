package controllers

import (
	"log"
	"shollu/database"
	"time"

	"github.com/gofiber/fiber/v2"
)

func SaveAbsenQR(c *fiber.Ctx) error {
	body := struct {
		MesinID string `json:"mesin_id"`
		QRCode  string `json:"qr_code"`
		EventID int    `json:"event_id"`
	}{}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if body.MesinID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "mesin_id is required"})
	}

	if body.QRCode == "" {
		return c.Status(400).JSON(fiber.Map{"error": "No QR code data provided"})
	}

	// Optimasi: Gunakan index di kolom qr_code untuk mempercepat pencarian
	var userID int
	var fullname string
	err := database.DB.QueryRow("SELECT id, fullname FROM peserta WHERE qr_code = ?", body.QRCode).Scan(&userID, &fullname)
	if err != nil {
		log.Println("QR Code not found in database:", err)
		return c.Status(404).JSON(fiber.Map{"error": "No matching QR code found"})
	}

	_, err = database.DB.Exec("INSERT INTO absensi (user_id, finger_id, jam, mesin_id, event_id) VALUES (?, ?, ?, ?, ?)",
		userID, body.QRCode, time.Now().UTC(), body.MesinID, body.EventID)
	if err != nil {
		log.Println("Error inserting attendance record:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save attendance record"})
	}

	return c.JSON(fiber.Map{
		"message":  "QR Code found and attendance recorded",
		"qr_code":  body.QRCode,
		"user_id":  userID,
		"fullname": fullname,
		"event_id": body.EventID,
	})
}
