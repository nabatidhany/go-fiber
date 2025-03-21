package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"shollu/database"
	"time"

	"github.com/gofiber/fiber/v2"
)

// func SaveAbsenQR(c *fiber.Ctx) error {
// 	body := struct {
// 		MesinID string `json:"mesin_id"`
// 		QRCode  string `json:"qr_code"`
// 		EventID int    `json:"event_id"`
// 	}{}

// 	if err := c.BodyParser(&body); err != nil {
// 		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
// 	}

// 	if body.MesinID == "" {
// 		return c.Status(400).JSON(fiber.Map{"error": "mesin_id is required"})
// 	}

// 	if body.QRCode == "" {
// 		return c.Status(400).JSON(fiber.Map{"error": "No QR code data provided"})
// 	}

// 	// Optimasi: Gunakan index di kolom qr_code untuk mempercepat pencarian
// 	var userID int
// 	var fullname string
// 	err := database.DB.QueryRow("SELECT id, fullname FROM peserta WHERE qr_code = ?", body.QRCode).Scan(&userID, &fullname)
// 	if err != nil {
// 		log.Println("QR Code not found in database:", err)
// 		return c.Status(404).JSON(fiber.Map{"error": "No matching QR code found"})
// 	}

// 	// Cek apakah user terdaftar dalam event yang sesuai
// 	var exists bool
// 	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM detail_peserta WHERE id_peserta = ? AND id_event = ?)", userID, body.EventID).Scan(&exists)
// 	if err != nil {
// 		log.Println("Error checking event participation:", err)
// 		return c.Status(500).JSON(fiber.Map{"error": "Database error while checking event participation"})
// 	}
// 	if !exists {
// 		return c.Status(404).JSON(fiber.Map{"error": "User is not registered for this event"})
// 	}

// 	_, err = database.DB.Exec("INSERT INTO absensi (user_id, finger_id, jam, mesin_id, event_id) VALUES (?, ?, ?, ?, ?)",
// 		userID, body.QRCode, time.Now().UTC(), body.MesinID, body.EventID)
// 	if err != nil {
// 		log.Println("Error inserting attendance record:", err)
// 		return c.Status(500).JSON(fiber.Map{"error": "Failed to save attendance record"})
// 	}

// 	return c.JSON(fiber.Map{
// 		"message":  "QR Code found and attendance recorded",
// 		"qr_code":  body.QRCode,
// 		"user_id":  userID,
// 		"fullname": fullname,
// 		"event_id": body.EventID,
// 	})
// }

// new version
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

	// Cek user berdasarkan QR Code
	var userID int
	var fullname string
	err := database.DB.QueryRow("SELECT id, fullname FROM peserta WHERE qr_code = ?", body.QRCode).Scan(&userID, &fullname)
	if err != nil {
		log.Println("QR Code not found in database:", err)
		return c.Status(404).JSON(fiber.Map{"error": "No matching QR code found"})
	}

	// Cek apakah user terdaftar dalam event
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM detail_peserta WHERE id_peserta = ? AND id_event = ?)", userID, body.EventID).Scan(&exists)
	if err != nil {
		log.Println("Error checking event participation:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Database error while checking event participation"})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "User is not registered for this event"})
	}

	// Default tag kosong
	tag := ""

	if body.EventID == 3 {
		// Ambil id_masjid dari tabel petugas
		var idMasjid int
		err = database.DB.QueryRow("SELECT id_masjid FROM petugas WHERE id_user = ?", body.MesinID).Scan(&idMasjid)
		if err != nil {
			log.Println("Masjid not found for the given MesinID:", err)
			return c.Status(404).JSON(fiber.Map{"error": "Masjid not found for this MesinID"})
		}

		// Ambil id_regional dari tabel masjid
		var idRegional int
		err = database.DB.QueryRow("SELECT regional_id FROM masjid WHERE id = ?", idMasjid).Scan(&idRegional)
		if err != nil {
			log.Println("Regional ID not found for Masjid:", err)
			return c.Status(404).JSON(fiber.Map{"error": "Regional ID not found for Masjid"})
		}

		// Ambil kode kota dari tabel regional
		var kotaCode string
		err = database.DB.QueryRow("SELECT code FROM regional WHERE id = ?", idRegional).Scan(&kotaCode)
		if err != nil {
			log.Println("Regional code not found:", err)
			return c.Status(404).JSON(fiber.Map{"error": "Regional code not found"})
		}

		// Ambil jadwal sholat dari API
		date := time.Now().Format("2006-01-02")
		apiURL := "https://api.myquran.com/v2/sholat/jadwal/" + kotaCode + "/" + date
		resp, err := http.Get(apiURL)
		if err != nil {
			log.Println("Error fetching prayer schedule:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch prayer schedule"})
		}
		defer resp.Body.Close()

		var result struct {
			Data struct {
				Jadwal map[string]string `json:"jadwal"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Println("Error decoding prayer schedule response:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to decode prayer schedule response"})
		}

		// Tentukan waktu sholat berdasarkan jam saat ini
		// Gunakan zona waktu WIB
		loc, _ := time.LoadLocation("Asia/Jakarta")
		currentTime := time.Now().In(loc)

		for prayer, prayerTime := range result.Data.Jadwal {
			// Gabungkan dengan tanggal hari ini sebelum parsing
			prayerDateTime, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+prayerTime, loc)

			startTime := prayerDateTime.Add(-30 * time.Minute)
			endTime := prayerDateTime.Add(30 * time.Minute)

			if currentTime.After(startTime) && currentTime.Before(endTime) {
				tag = prayer
				break
			}
		}

		if tag == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Absensi hanya diperbolehkan dalam rentang 30 menit sebelum dan sesudah waktu sholat"})
		}
	}

	// Simpan absensi
	_, err = database.DB.Exec("INSERT INTO absensi (user_id, finger_id, jam, mesin_id, event_id, tag) VALUES (?, ?, ?, ?, ?, ?)",
		userID, body.QRCode, time.Now().UTC(), body.MesinID, body.EventID, tag)
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
		"tag":      tag,
	})
}
