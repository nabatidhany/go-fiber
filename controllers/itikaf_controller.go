package controllers

import (
	"database/sql"
	"net/http"
	"shollu/database"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Struct untuk response rekap absensi
type RekapAbsen struct {
	UserID   int       `json:"user_id"`
	Fullname string    `json:"fullname"`
	Jam      time.Time `json:"jam"`
}

// Struct untuk menyimpan informasi masjid
type MasjidInfo struct {
	Nama   string `json:"name"`
	Alamat string `json:"alamat"`
	Foto   string `json:"foto"`
}

// Handler untuk mendapatkan rekap absen berdasarkan filter tanggal
func GetRekapAbsen(c *fiber.Ctx) error {
	idMasjid := c.Params("id_masjid") // Ambil id_masjid dari parameter URL
	idEvent := c.Query("id_event")    // Ambil id_event dari query parameter
	tanggal := c.Query("tanggal")     // Ambil tanggal dari query parameter

	// Gunakan tanggal hari ini jika tidak ada query parameter tanggal
	if tanggal == "" {
		tanggal = time.Now().Format("2006-01-02")
	}

	var query string
	var args []interface{}

	// Ambil informasi masjid
	var masjid MasjidInfo
	err := database.DB.QueryRow(`
		SELECT nama, alamat, foto FROM masjid WHERE id = ?
	`, idMasjid).Scan(&masjid.Nama, &masjid.Alamat, &masjid.Foto)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Masjid not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch masjid data"})
	}

	switch idEvent {
	case "1":
		query = `
			SELECT absensi.user_id, COALESCE(peserta.fullname, '') AS fullname, absensi.created_at AS jam
			FROM absensi
			LEFT JOIN petugas ON absensi.mesin_id = petugas.id
			LEFT JOIN peserta ON absensi.user_id = peserta.id
			WHERE absensi.event_id = ? AND petugas.id_masjid = ?
			AND DATE(absensi.jam) = DATE(?)`
		args = append(args, idEvent, idMasjid, tanggal)
	case "2":
		query = `
			SELECT absensi.user_id, COALESCE(peserta.fullname, '') AS fullname, absensi.created_at AS jam
			FROM absensi
			LEFT JOIN petugas ON absensi.mesin_id = petugas.id
			LEFT JOIN peserta ON absensi.user_id = peserta.id
			WHERE absensi.event_id = ? AND petugas.id_masjid = ?
			AND DATE(absensi.jam) = DATE(?)`
		args = append(args, idEvent, idMasjid, tanggal)
	case "3":
		jamMin := c.Query("jam_min") // Ambil jam_min dari query parameter
		jamMax := c.Query("jam_max") // Ambil jam_max dari query parameter

		// Load lokasi zona waktu Jakarta
		locJakarta, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load timezone"})
		}

		// Konversi jam_min dan jam_max dari WIB ke UTC
		if jamMin != "" && jamMax != "" {
			tanggalWIB := tanggal + " " + jamMin
			jamMinTime, err := time.ParseInLocation("2006-01-02 15:04:05", tanggalWIB, locJakarta)
			if err != nil {
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid jam_min format"})
			}

			tanggalWIB = tanggal + " " + jamMax
			jamMaxTime, err := time.ParseInLocation("2006-01-02 15:04:05", tanggalWIB, locJakarta)
			if err != nil {
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid jam_max format"})
			}

			// Konversi ke UTC
			jamMinUTC := jamMinTime.UTC().Format("15:04:05")
			jamMaxUTC := jamMaxTime.UTC().Format("15:04:05")

			jamMin = jamMinUTC
			jamMax = jamMaxUTC
		}

		query = `
			SELECT absensi.user_id, COALESCE(peserta.fullname, '') AS fullname, absensi.created_at AS jam
			FROM absensi
			LEFT JOIN petugas ON absensi.mesin_id = petugas.id
			LEFT JOIN peserta ON absensi.user_id = peserta.id
			WHERE absensi.event_id = ? AND petugas.id_masjid = ?
			AND DATE(absensi.created_at) = DATE(?)
			AND TIME(absensi.created_at) BETWEEN ? AND ?`
		args = append(args, idEvent, idMasjid, tanggal, jamMin, jamMax)
	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid event_id"})
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch rekap absen"})
	}
	defer rows.Close()

	var rekapList []RekapAbsen

	for rows.Next() {
		var rekap RekapAbsen
		if err := rows.Scan(&rekap.UserID, &rekap.Fullname, &rekap.Jam); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		rekapList = append(rekapList, rekap)
	}

	for i := range rekapList {
		rekapList[i].Jam = rekapList[i].Jam.UTC() // Pastikan UTC
	}

	if len(rekapList) == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"message": "No attendance records found"})
	}

	return c.JSON(fiber.Map{
		"message": "Success",
		"masjid":  masjid,
		"data":    rekapList,
	})
}
