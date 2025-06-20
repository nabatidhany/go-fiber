package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shollu/database"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/gofiber/fiber/v2"
)

type cachedJadwal struct {
	Jadwal map[string]string
	Expiry time.Time
	Date   string
}

var (
	jadwalCache       = make(map[string]cachedJadwal)
	currentCachedDate string
)

var localCache = cache.New(30*time.Minute, 10*time.Minute)

func clearOldCacheIfNeeded() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	today := time.Now().In(loc).Format("2006-01-02")

	if currentCachedDate != "" && currentCachedDate != today {
		jadwalCache = make(map[string]cachedJadwal)
	}
	currentCachedDate = today
}

// version with caching
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

	type Peserta struct {
		ID       int
		Fullname string
	}

	var peserta Peserta
	if cached, found := localCache.Get("peserta:" + body.QRCode); found {
		peserta = cached.(Peserta)
	} else {
		err := database.DB.QueryRow("SELECT id, fullname FROM peserta WHERE qr_code = ?", body.QRCode).Scan(&peserta.ID, &peserta.Fullname)
		if err != nil {
			log.Println("QR Code not found in database:", err)
			return c.Status(404).JSON(fiber.Map{"error": "No matching QR code found"})
		}
		localCache.Set("peserta:"+body.QRCode, peserta, cache.DefaultExpiration)
	}
	userID := peserta.ID
	fullname := peserta.Fullname

	eventKey := fmt.Sprintf("event:%d:%d", userID, body.EventID)
	var exists bool
	if cached, found := localCache.Get(eventKey); found {
		exists = cached.(bool)
	} else {
		err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM detail_peserta WHERE id_peserta = ? AND id_event = ?)", userID, body.EventID).Scan(&exists)
		if err != nil {
			log.Println("Error checking event participation:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Database error while checking event participation"})
		}
		localCache.Set(eventKey, exists, cache.DefaultExpiration)
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "User is not registered for this event"})
	}

	tag := ""

	if body.EventID == 3 {
		masjidKey := "masjid:" + body.MesinID
		var idMasjid int
		if cached, found := localCache.Get(masjidKey); found {
			idMasjid = cached.(int)
		} else {
			err := database.DB.QueryRow("SELECT id_masjid FROM petugas WHERE id_user = ?", body.MesinID).Scan(&idMasjid)
			if err != nil {
				log.Println("Masjid not found for the given MesinID:", err)
				return c.Status(404).JSON(fiber.Map{"error": "Masjid not found for this MesinID"})
			}
			localCache.Set(masjidKey, idMasjid, cache.DefaultExpiration)
		}

		regionalKey := fmt.Sprintf("regional_id:%d", idMasjid)
		var idRegional int
		if cached, found := localCache.Get(regionalKey); found {
			idRegional = cached.(int)
		} else {
			err := database.DB.QueryRow("SELECT regional_id FROM masjid WHERE id = ?", idMasjid).Scan(&idRegional)
			if err != nil {
				log.Println("Regional ID not found for Masjid:", err)
				return c.Status(404).JSON(fiber.Map{"error": "Regional ID not found for Masjid"})
			}
			localCache.Set(regionalKey, idRegional, cache.DefaultExpiration)
		}

		codeKey := fmt.Sprintf("regcode:%d", idRegional)
		var kotaCode string
		if cached, found := localCache.Get(codeKey); found {
			kotaCode = cached.(string)
		} else {
			err := database.DB.QueryRow("SELECT code FROM regional WHERE id = ?", idRegional).Scan(&kotaCode)
			if err != nil {
				log.Println("Regional code not found:", err)
				return c.Status(404).JSON(fiber.Map{"error": "Regional code not found"})
			}
			localCache.Set(codeKey, kotaCode, cache.DefaultExpiration)
		}

		clearOldCacheIfNeeded()

		loc, _ := time.LoadLocation("Asia/Jakarta")
		now := time.Now().In(loc)
		date := now.Format("2006-01-02")
		key := kotaCode + "-" + date

		var jadwal map[string]string

		cached, found := jadwalCache[key]
		if found && time.Now().Before(cached.Expiry) {
			jadwal = cached.Jadwal
		} else {
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

			jadwal = result.Data.Jadwal
			jadwalCache[key] = cachedJadwal{
				Jadwal: jadwal,
				Expiry: time.Now().Add(6 * time.Hour),
				Date:   date,
			}
		}

		currentTime := now

		validPrayers := map[string]bool{
			"subuh":   true,
			"dzuhur":  true,
			"ashar":   true,
			"maghrib": true,
			"isya":    true,
		}

		configs := make(map[string]struct {
			Before time.Duration
			After  time.Duration
		})

		rows, err := database.DB.Query("SELECT nama_sholat, sebelum_menit, sesudah_menit FROM sholat_config")
		if err != nil {
			log.Println("Failed to fetch sholat configs:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Database error while fetching sholat configs"})
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			var before, after int
			if err := rows.Scan(&name, &before, &after); err != nil {
				continue
			}
			configs[strings.ToLower(name)] = struct {
				Before time.Duration
				After  time.Duration
			}{
				Before: time.Duration(before) * time.Minute,
				After:  time.Duration(after) * time.Minute,
			}
		}

		for prayer, prayerTime := range jadwal {
			lowerPrayer := strings.ToLower(prayer)
			conf, ok := configs[lowerPrayer]
			if !ok || !validPrayers[lowerPrayer] {
				continue
			}

			prayerDateTime, err := time.ParseInLocation("2006-01-02 15:04", date+" "+prayerTime, loc)
			if err != nil {
				log.Println("Failed to parse prayer time:", prayer, prayerTime)
				continue
			}

			startTime := prayerDateTime.Add(-conf.Before)
			endTime := prayerDateTime.Add(conf.After)

			if currentTime.After(startTime) && currentTime.Before(endTime) {
				tag = lowerPrayer
				break
			}
		}

		if tag == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Absensi hanya diperbolehkan dalam rentang waktu yang telah ditentukan untuk sholat"})
		}

		var alreadyExists bool
		err = database.DB.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM absensi WHERE user_id = ? AND event_id = ? AND tag = ? AND DATE(CONVERT_TZ(created_at, '+00:00', '+07:00')) = ? )`,
			userID, body.EventID, tag, date,
		).Scan(&alreadyExists)
		if err != nil {
			log.Println("Error checking existing attendance:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Database error while checking existing attendance"})
		}

		if alreadyExists {
			return c.Status(400).JSON(fiber.Map{"error": "User sudah absen untuk sholat " + strings.Title(tag)})
		}

		var count int
		err = database.DB.QueryRow(`
			SELECT COUNT(*) FROM absensi 
			WHERE event_id = ? AND tag = ? AND DATE(CONVERT_TZ(created_at, '+00:00', '+07:00')) = ? and mesin_id = ?
		`, body.EventID, tag, date, body.MesinID).Scan(&count)
		if err != nil {
			log.Println("Error counting attendance:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to count attendance"})
		}
		urutanKehadiran := count + 1

		pointSholat := 0
		switch strings.ToLower(tag) {
		case "subuh":
			pointSholat = 40
		case "maghrib", "magrib":
			pointSholat = 30
		case "isya":
			pointSholat = 30
		}

		pointHadir := 0
		if urutanKehadiran <= 10 {
			pointHadir = 11 - urutanKehadiran
		}

		totalPoint := pointSholat + pointHadir

		_, err = database.DB.Exec(`
			INSERT INTO poin (user_id, tanggal, tag, point_sholat, point_kehadiran, total_point)
			VALUES (?, ?, ?, ?, ?, ?)`,
			userID, date, tag, pointSholat, pointHadir, totalPoint)
		if err != nil {
			log.Println("Error inserting point:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to save point"})
		}
	}

	_, err := database.DB.Exec("INSERT INTO absensi (user_id, finger_id, jam, mesin_id, event_id, tag) VALUES (?, ?, ?, ?, ?, ?)",
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

// 	var userID int
// 	var fullname string
// 	err := database.DB.QueryRow("SELECT id, fullname FROM peserta WHERE qr_code = ?", body.QRCode).Scan(&userID, &fullname)
// 	if err != nil {
// 		log.Println("QR Code not found in database:", err)
// 		return c.Status(404).JSON(fiber.Map{"error": "No matching QR code found"})
// 	}

// 	var exists bool
// 	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM detail_peserta WHERE id_peserta = ? AND id_event = ?)", userID, body.EventID).Scan(&exists)
// 	if err != nil {
// 		log.Println("Error checking event participation:", err)
// 		return c.Status(500).JSON(fiber.Map{"error": "Database error while checking event participation"})
// 	}
// 	if !exists {
// 		return c.Status(404).JSON(fiber.Map{"error": "User is not registered for this event"})
// 	}

// 	tag := ""

// 	if body.EventID == 3 {
// 		var idMasjid int
// 		err = database.DB.QueryRow("SELECT id_masjid FROM petugas WHERE id_user = ?", body.MesinID).Scan(&idMasjid)
// 		if err != nil {
// 			log.Println("Masjid not found for the given MesinID:", err)
// 			return c.Status(404).JSON(fiber.Map{"error": "Masjid not found for this MesinID"})
// 		}

// 		var idRegional int
// 		err = database.DB.QueryRow("SELECT regional_id FROM masjid WHERE id = ?", idMasjid).Scan(&idRegional)
// 		if err != nil {
// 			log.Println("Regional ID not found for Masjid:", err)
// 			return c.Status(404).JSON(fiber.Map{"error": "Regional ID not found for Masjid"})
// 		}

// 		var kotaCode string
// 		err = database.DB.QueryRow("SELECT code FROM regional WHERE id = ?", idRegional).Scan(&kotaCode)
// 		if err != nil {
// 			log.Println("Regional code not found:", err)
// 			return c.Status(404).JSON(fiber.Map{"error": "Regional code not found"})
// 		}

// 		clearOldCacheIfNeeded()

// 		loc, _ := time.LoadLocation("Asia/Jakarta")
// 		now := time.Now().In(loc)
// 		date := now.Format("2006-01-02")
// 		key := kotaCode + "-" + date

// 		var jadwal map[string]string

// 		cached, found := jadwalCache[key]
// 		if found && time.Now().Before(cached.Expiry) {
// 			jadwal = cached.Jadwal
// 		} else {
// 			apiURL := "https://api.myquran.com/v2/sholat/jadwal/" + kotaCode + "/" + date
// 			resp, err := http.Get(apiURL)
// 			if err != nil {
// 				log.Println("Error fetching prayer schedule:", err)
// 				return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch prayer schedule"})
// 			}
// 			defer resp.Body.Close()

// 			var result struct {
// 				Data struct {
// 					Jadwal map[string]string `json:"jadwal"`
// 				} `json:"data"`
// 			}

// 			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// 				log.Println("Error decoding prayer schedule response:", err)
// 				return c.Status(500).JSON(fiber.Map{"error": "Failed to decode prayer schedule response"})
// 			}

// 			jadwal = result.Data.Jadwal
// 			jadwalCache[key] = cachedJadwal{
// 				Jadwal: jadwal,
// 				Expiry: time.Now().Add(6 * time.Hour),
// 				Date:   date,
// 			}
// 		}

// 		currentTime := now

// 		validPrayers := map[string]bool{
// 			"subuh":   true,
// 			"dzuhur":  true,
// 			"ashar":   true,
// 			"maghrib": true,
// 			"isya":    true,
// 		}

// 		configs := make(map[string]struct {
// 			Before time.Duration
// 			After  time.Duration
// 		})

// 		rows, err := database.DB.Query("SELECT nama_sholat, sebelum_menit, sesudah_menit FROM sholat_config")
// 		if err != nil {
// 			log.Println("Failed to fetch sholat configs:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Database error while fetching sholat configs"})
// 		}
// 		defer rows.Close()

// 		for rows.Next() {
// 			var name string
// 			var before, after int
// 			if err := rows.Scan(&name, &before, &after); err != nil {
// 				continue
// 			}
// 			configs[strings.ToLower(name)] = struct {
// 				Before time.Duration
// 				After  time.Duration
// 			}{
// 				Before: time.Duration(before) * time.Minute,
// 				After:  time.Duration(after) * time.Minute,
// 			}
// 		}

// 		for prayer, prayerTime := range jadwal {
// 			lowerPrayer := strings.ToLower(prayer)
// 			conf, ok := configs[lowerPrayer]
// 			if !ok || !validPrayers[lowerPrayer] {
// 				continue
// 			}

// 			prayerDateTime, err := time.ParseInLocation("2006-01-02 15:04", date+" "+prayerTime, loc)
// 			if err != nil {
// 				log.Println("Failed to parse prayer time:", prayer, prayerTime)
// 				continue
// 			}

// 			startTime := prayerDateTime.Add(-conf.Before)
// 			endTime := prayerDateTime.Add(conf.After)

// 			if currentTime.After(startTime) && currentTime.Before(endTime) {
// 				tag = lowerPrayer
// 				break
// 			}
// 		}

// 		if tag == "" {
// 			return c.Status(400).JSON(fiber.Map{"error": "Absensi hanya diperbolehkan dalam rentang waktu yang telah ditentukan untuk sholat"})
// 		}

// 		var alreadyExists bool
// 		err = database.DB.QueryRow(
// 			`SELECT EXISTS(SELECT 1 FROM absensi WHERE user_id = ? AND event_id = ? AND tag = ? AND DATE(CONVERT_TZ(created_at, '+00:00', '+07:00')) = ? )`,
// 			userID, body.EventID, tag, date,
// 		).Scan(&alreadyExists)
// 		if err != nil {
// 			log.Println("Error checking existing attendance:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Database error while checking existing attendance"})
// 		}

// 		if alreadyExists {
// 			return c.Status(400).JSON(fiber.Map{"error": "User sudah absen untuk sholat " + strings.Title(tag)})
// 		}

// 		var count int
// 		err = database.DB.QueryRow(`
// 			SELECT COUNT(*) FROM absensi
// 			WHERE event_id = ? AND tag = ? AND DATE(CONVERT_TZ(created_at, '+00:00', '+07:00')) = ? and mesin_id = ?
// 		`, body.EventID, tag, date, body.MesinID).Scan(&count)
// 		if err != nil {
// 			log.Println("Error counting attendance:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Failed to count attendance"})
// 		}
// 		urutanKehadiran := count + 1

// 		pointSholat := 0
// 		switch strings.ToLower(tag) {
// 		case "subuh":
// 			pointSholat = 40
// 		case "maghrib", "magrib":
// 			pointSholat = 30
// 		case "isya":
// 			pointSholat = 30
// 		}

// 		pointHadir := 0
// 		if urutanKehadiran <= 10 {
// 			pointHadir = 11 - urutanKehadiran
// 		}

// 		totalPoint := pointSholat + pointHadir

// 		_, err = database.DB.Exec(`
// 			INSERT INTO poin (user_id, tanggal, tag, point_sholat, point_kehadiran, total_point)
// 			VALUES (?, ?, ?, ?, ?, ?)`,
// 			userID, date, tag, pointSholat, pointHadir, totalPoint)
// 		if err != nil {
// 			log.Println("Error inserting point:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Failed to save point"})
// 		}
// 	}

// 	_, err = database.DB.Exec("INSERT INTO absensi (user_id, finger_id, jam, mesin_id, event_id, tag) VALUES (?, ?, ?, ?, ?, ?)",
// 		userID, body.QRCode, time.Now().UTC(), body.MesinID, body.EventID, tag)
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
// 		"tag":      tag,
// 	})
// }

// new version
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

// 	// Cek user berdasarkan QR Code
// 	var userID int
// 	var fullname string
// 	err := database.DB.QueryRow("SELECT id, fullname FROM peserta WHERE qr_code = ?", body.QRCode).Scan(&userID, &fullname)
// 	if err != nil {
// 		log.Println("QR Code not found in database:", err)
// 		return c.Status(404).JSON(fiber.Map{"error": "No matching QR code found"})
// 	}

// 	// Cek apakah user terdaftar dalam event
// 	var exists bool
// 	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM detail_peserta WHERE id_peserta = ? AND id_event = ?)", userID, body.EventID).Scan(&exists)
// 	if err != nil {
// 		log.Println("Error checking event participation:", err)
// 		return c.Status(500).JSON(fiber.Map{"error": "Database error while checking event participation"})
// 	}
// 	if !exists {
// 		return c.Status(404).JSON(fiber.Map{"error": "User is not registered for this event"})
// 	}

// 	// Default tag kosong
// 	tag := ""

// 	if body.EventID == 3 {
// 		// Ambil id_masjid dari tabel petugas
// 		var idMasjid int
// 		err = database.DB.QueryRow("SELECT id_masjid FROM petugas WHERE id_user = ?", body.MesinID).Scan(&idMasjid)
// 		if err != nil {
// 			log.Println("Masjid not found for the given MesinID:", err)
// 			return c.Status(404).JSON(fiber.Map{"error": "Masjid not found for this MesinID"})
// 		}

// 		// Ambil id_regional dari tabel masjid
// 		var idRegional int
// 		err = database.DB.QueryRow("SELECT regional_id FROM masjid WHERE id = ?", idMasjid).Scan(&idRegional)
// 		if err != nil {
// 			log.Println("Regional ID not found for Masjid:", err)
// 			return c.Status(404).JSON(fiber.Map{"error": "Regional ID not found for Masjid"})
// 		}

// 		// Ambil kode kota dari tabel regional
// 		var kotaCode string
// 		err = database.DB.QueryRow("SELECT code FROM regional WHERE id = ?", idRegional).Scan(&kotaCode)
// 		if err != nil {
// 			log.Println("Regional code not found:", err)
// 			return c.Status(404).JSON(fiber.Map{"error": "Regional code not found"})
// 		}

// 		// Ambil jadwal sholat dari API
// 		date := time.Now().Format("2006-01-02")
// 		apiURL := "https://api.myquran.com/v2/sholat/jadwal/" + kotaCode + "/" + date
// 		resp, err := http.Get(apiURL)
// 		if err != nil {
// 			log.Println("Error fetching prayer schedule:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch prayer schedule"})
// 		}
// 		defer resp.Body.Close()

// 		var result struct {
// 			Data struct {
// 				Jadwal map[string]string `json:"jadwal"`
// 			} `json:"data"`
// 		}

// 		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// 			log.Println("Error decoding prayer schedule response:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Failed to decode prayer schedule response"})
// 		}

// 		// Tentukan waktu sholat berdasarkan jam saat ini
// 		// Gunakan zona waktu WIB
// 		loc, _ := time.LoadLocation("Asia/Jakarta")
// 		currentTime := time.Now().In(loc)

// 		// Daftar jadwal sholat yang diperbolehkan
// 		validPrayers := map[string]bool{
// 			"subuh":   true,
// 			"dzuhur":  true,
// 			"ashar":   true,
// 			"maghrib": true,
// 			"isya":    true,
// 		}

// 		// Ambil konfigurasi rentang waktu dari database
// 		configs := make(map[string]struct {
// 			Before time.Duration
// 			After  time.Duration
// 		})

// 		rows, err := database.DB.Query("SELECT nama_sholat, sebelum_menit, sesudah_menit FROM sholat_config")
// 		if err != nil {
// 			log.Println("Failed to fetch sholat configs:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Database error while fetching sholat configs"})
// 		}
// 		defer rows.Close()

// 		for rows.Next() {
// 			var name string
// 			var before, after int
// 			if err := rows.Scan(&name, &before, &after); err != nil {
// 				continue
// 			}
// 			configs[strings.ToLower(name)] = struct {
// 				Before time.Duration
// 				After  time.Duration
// 			}{
// 				Before: time.Duration(before) * time.Minute,
// 				After:  time.Duration(after) * time.Minute,
// 			}
// 		}

// 		for prayer, prayerTime := range result.Data.Jadwal {
// 			lowerPrayer := strings.ToLower(prayer)
// 			conf, ok := configs[lowerPrayer]
// 			if !ok {
// 				continue // skip jika tidak ada config-nya
// 			}
// 			// Hanya proses jadwal sholat yang valid
// 			if !validPrayers[lowerPrayer] {
// 				continue
// 			}

// 			prayerDateTime, err := time.ParseInLocation("2006-01-02 15:04", date+" "+prayerTime, loc)
// 			if err != nil {
// 				log.Println("Failed to parse prayer time:", prayer, prayerTime)
// 				continue
// 			}

// 			// startTime := prayerDateTime.Add(-30 * time.Minute)
// 			// endTime := prayerDateTime.Add(30 * time.Minute)

// 			startTime := prayerDateTime.Add(-conf.Before)
// 			endTime := prayerDateTime.Add(conf.After)

// 			if currentTime.After(startTime) && currentTime.Before(endTime) {
// 				tag = lowerPrayer
// 				break
// 			}
// 		}

// 		// for prayer, prayerTime := range result.Data.Jadwal {
// 		// 	// Gabungkan dengan tanggal hari ini sebelum parsing
// 		// 	prayerDateTime, _ := time.ParseInLocation("2006-01-02 15:04", date+" "+prayerTime, loc)

// 		// 	startTime := prayerDateTime.Add(-30 * time.Minute)
// 		// 	endTime := prayerDateTime.Add(30 * time.Minute)

// 		// 	if currentTime.After(startTime) && currentTime.Before(endTime) {
// 		// 		tag = prayer
// 		// 		break
// 		// 	}
// 		// }

// 		if tag == "" {
// 			return c.Status(400).JSON(fiber.Map{"error": "Absensi hanya diperbolehkan dalam rentang 60 menit sebelum dan sesudah waktu sholat"})
// 		}

// 		// Validasi: jika user sudah absen di event dan tag yang sama
// 		if tag != "" {
// 			var alreadyExists bool
// 			err = database.DB.QueryRow(
// 				`SELECT EXISTS(SELECT 1 FROM absensi WHERE user_id = ? AND event_id = ? AND tag = ? AND DATE(CONVERT_TZ(created_at, '+00:00', '+07:00')) = ? )`,
// 				userID, body.EventID, tag, date,
// 			).Scan(&alreadyExists)
// 			if err != nil {
// 				log.Println("Error checking existing attendance:", err)
// 				return c.Status(500).JSON(fiber.Map{"error": "Database error while checking existing attendance"})
// 			}

// 			if alreadyExists {
// 				return c.Status(400).JSON(fiber.Map{"error": "User sudah absen untuk sholat " + strings.Title(tag)})
// 			}
// 		}

// 		// 4. Hitung urutan kehadiran
// 		var count int
// 		err = database.DB.QueryRow(`
// 			SELECT COUNT(*) FROM absensi
// 			WHERE event_id = ? AND tag = ? AND DATE(CONVERT_TZ(created_at, '+00:00', '+07:00')) = ? and mesin_id = ?
// 		`, body.EventID, tag, date, body.MesinID).Scan(&count)
// 		if err != nil {
// 			log.Println("Error counting attendance:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Failed to count attendance"})
// 		}
// 		urutanKehadiran := count + 1

// 		// 5. Hitung point sholat
// 		pointSholat := 0
// 		switch strings.ToLower(tag) {
// 		case "subuh":
// 			pointSholat = 40
// 		case "maghrib", "magrib":
// 			pointSholat = 30
// 		case "isya":
// 			pointSholat = 30
// 		}

// 		// 6. Hitung point kehadiran
// 		pointHadir := 0
// 		if urutanKehadiran <= 10 {
// 			pointHadir = 11 - urutanKehadiran
// 		}

// 		totalPoint := pointSholat + pointHadir

// 		// Simpan poin ke tabel `poin`
// 		_, err = database.DB.Exec(`
// 			INSERT INTO poin (user_id, tanggal, tag, point_sholat, point_kehadiran, total_point)
// 			VALUES (?, ?, ?, ?, ?, ?)`,
// 			userID, date, tag, pointSholat, pointHadir, totalPoint)
// 		if err != nil {
// 			log.Println("Error inserting point:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Failed to save point"})
// 		}
// 	}

// 	// Simpan absensi
// 	_, err = database.DB.Exec("INSERT INTO absensi (user_id, finger_id, jam, mesin_id, event_id, tag) VALUES (?, ?, ?, ?, ?, ?)",
// 		userID, body.QRCode, time.Now().UTC(), body.MesinID, body.EventID, tag)
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
// 		"tag":      tag,
// 	})
// }
