package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"shollu/database"
	"shollu/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Struct untuk request registrasi peserta
type RegisterPesertaRequest struct {
	FullName   string `json:"fullname" validate:"required,min=3"`
	Contact    string `json:"contact" validate:"required,startswith=0,min=10,max=12,numeric"`
	Gender     string `json:"gender" validate:"required,oneof=male female"`
	Dob        string `json:"dob" validate:"required"`
	MasjidID   int    `json:"masjid_id" validate:"required,min=1"`
	IsHideName bool   `json:"isHideName"`
	QRCode     string `json:"qrCode" validate:"omitempty,len=12"`
	EventID    int    `json:"event_id" validate:"omitempty,min=1"`
}

func GenerateRandomID() string {
	b := make([]byte, 6) // 6 byte = 12 karakter hex
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Handler untuk registrasi peserta
func RegisterPesertaItikaf(c *fiber.Ctx) error {
	// Parse body JSON
	var req RegisterPesertaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Validasi input
	if err := utils.Validate.Struct(req); err != nil {
		errors := utils.FormatValidationErrors(err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}

	// Konversi DOB ke format time.Time
	dob, err := time.Parse("2006-01-02", req.Dob)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date format. Use YYYY-MM-DD"})
	}

	// Cek apakah nomor HP sudah terdaftar
	var exists int
	err = database.DB.QueryRow("SELECT COUNT(*) FROM peserta WHERE contact = ?", req.Contact).Scan(&exists)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}
	if exists > 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Nomor HP sudah terdaftar"})
	}

	// Gunakan QR Code dari request jika diberikan, atau generate yang baru
	qrCode := req.QRCode
	if qrCode == "" {
		qrCode = GenerateRandomID()
	} else {
		// Periksa apakah QR Code sudah ada di database
		var qrExists int
		err = database.DB.QueryRow("SELECT COUNT(*) FROM peserta WHERE qr_code = ?", qrCode).Scan(&qrExists)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Database error"})
		}
		if qrExists > 0 {
			return c.Status(400).JSON(fiber.Map{"error": "QR Code sudah digunakan"})
		}
	}

	// Gunakan Event ID dari request jika diberikan, atau gunakan default 2
	eventID := req.EventID
	if eventID == 0 {
		eventID = 2
	}

	// Insert ke `peserta`
	result, err := database.DB.Exec("INSERT INTO peserta (fullname, contact, gender, dob, masjid_id, isHideName, qr_code, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		req.FullName, req.Contact, req.Gender, dob, req.MasjidID, req.IsHideName, qrCode, 1)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to insert peserta"})
	}

	// Ambil ID peserta yang baru saja dibuat
	idPeserta, err := result.LastInsertId()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve peserta ID"})
	}

	// Insert ke `detail_peserta`
	_, err = database.DB.Exec("INSERT INTO detail_peserta (id_peserta, id_event, status) VALUES (?, ?, ?)", idPeserta, eventID, 1)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to insert detail peserta"})
	}

	// return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Peserta registered successfully"})
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message":    "Peserta registered successfully",
		"qr_code":    qrCode,
		"is_peserta": idPeserta,
	})
}
