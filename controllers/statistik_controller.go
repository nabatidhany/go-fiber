package controllers

import (
	"fmt"
	"log"
	"shollu/database"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func GetNewRegistrantStatistics(c *fiber.Ctx) error {
	// Ambil event_id dari query parameter
	eventIDStr := c.Query("event_id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil || eventID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "event_id is required"})
	}
	eventDate := c.Query("event_date")

	masjidStats := []map[string]interface{}{}
	rows, err := database.DB.Query(`
			SELECT
    m.id AS masjid_id,
    m.nama AS masjid_nama,
    m.alamat,
    COALESCE(COUNT(DISTINCT peserta.id), 0) AS total_count
		FROM masjid m
		LEFT JOIN peserta ON m.id = peserta.masjid_id
				AND DATE(CONVERT_TZ(peserta.created_at, '+00:00', '+07:00')) = DATE(?)
		left JOIN setting on setting.id_masjid = m.id
		where setting.id_event = ?
		GROUP BY m.id, m.nama
		ORDER BY total_count DESC
	`, eventDate, eventID)

	if err != nil {
		log.Println("Error fetching masjid registrant statistics:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch masjid registrant statistics"})
	}
	defer rows.Close()

	for rows.Next() {
		var masjidID int
		var masjidNama string
		var masjidAlamat string
		var totalCount int

		if err := rows.Scan(&masjidID, &masjidNama, &masjidAlamat, &totalCount); err != nil {
			log.Println("Error scanning masjid row:", err)
			continue
		}

		masjidStats = append(masjidStats, map[string]interface{}{
			"masjid_id":     masjidID,
			"masjid_nama":   masjidNama,
			"masjid_alamat": masjidAlamat,
			"total_count":   totalCount,
		})
	}

	// Return response JSON
	return c.JSON(fiber.Map{
		"event_id":     eventID,
		"event_date":   eventDate,
		"masjid_stats": masjidStats,
	})

}

// func GetNewRegistrantStatistics(c *fiber.Ctx) error {
// 	// Ambil event_id dari query parameter
// 	eventIDStr := c.Query("event_id")
// 	eventID, err := strconv.Atoi(eventIDStr)
// 	if err != nil || eventID == 0 {
// 		return c.Status(400).JSON(fiber.Map{"error": "event_id is required"})
// 	}
// 	eventDate := c.Query("event_date")

// 	// Query berbeda jika event_id == 2
// 	var query string
// 	var queryParams []interface{}

// 	if eventID == 2 {
// 		query = `
// 			SELECT
// 				m.id AS masjid_id,
// 				m.nama AS masjid_nama,
// 				m.alamat,
// 				COALESCE(COUNT(DISTINCT peserta.id), 0) AS total_count
// 			FROM masjid m
// 			LEFT JOIN peserta ON m.id = peserta.masjid_id
// 				AND (
// 					CONVERT_TZ(peserta.created_at, '+00:00', '+07:00')
// 					BETWEEN CONCAT(?, ' 19:00:00')
// 					AND CONCAT(DATE_ADD(?, INTERVAL 1 DAY), ' 06:00:00')
// 				)
// 			LEFT JOIN setting ON setting.id_masjid = m.id
// 			WHERE setting.id_event = ?
// 			GROUP BY m.id, m.nama, m.alamat
// 			ORDER BY total_count DESC;
// 		`
// 		queryParams = []interface{}{eventDate, eventDate, eventID}
// 	} else {
// 		query = `
// 			SELECT
// 				m.id AS masjid_id,
// 				m.nama AS masjid_nama,
// 				m.alamat,
// 				COALESCE(COUNT(DISTINCT peserta.id), 0) AS total_count
// 			FROM masjid m
// 			LEFT JOIN peserta ON m.id = peserta.masjid_id
// 				AND DATE(CONVERT_TZ(peserta.created_at, '+00:00', '+07:00')) = DATE(?)
// 			LEFT JOIN setting ON setting.id_masjid = m.id
// 			WHERE setting.id_event = ?
// 			GROUP BY m.id, m.nama, m.alamat
// 			ORDER BY total_count DESC;
// 		`
// 		queryParams = []interface{}{eventDate, eventID}
// 	}

// 	masjidStats := []map[string]interface{}{}
// 	rows, err := database.DB.Query(query, queryParams...)
// 	if err != nil {
// 		log.Println("Error fetching masjid registrant statistics:", err)
// 		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch masjid registrant statistics"})
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var masjidID int
// 		var masjidNama string
// 		var masjidAlamat string
// 		var totalCount int

// 		if err := rows.Scan(&masjidID, &masjidNama, &masjidAlamat, &totalCount); err != nil {
// 			log.Println("Error scanning masjid row:", err)
// 			continue
// 		}

// 		masjidStats = append(masjidStats, map[string]interface{}{
// 			"masjid_id":     masjidID,
// 			"masjid_nama":   masjidNama,
// 			"masjid_alamat": masjidAlamat,
// 			"total_count":   totalCount,
// 		})
// 	}

// 	// Return response JSON
// 	return c.JSON(fiber.Map{
// 		"event_id":     eventID,
// 		"event_date":   eventDate,
// 		"masjid_stats": masjidStats,
// 	})
// }

// func GetEventStatistics(c *fiber.Ctx) error {
// 	// Ambil event_id dari query parameter
// 	eventIDStr := c.Query("event_id")
// 	eventID, err := strconv.Atoi(eventIDStr)
// 	if err != nil || eventID == 0 {
// 		return c.Status(400).JSON(fiber.Map{"error": "event_id is required"})
// 	}

// 	// Variabel untuk menyimpan hasil query
// 	var totalPeserta, totalAbsen, totalMale, totalFemale int
// 	var persenHadir float64

// 	// Query untuk mengambil statistik utama
// 	query := `
// 		SELECT
// 			COALESCE(total_peserta, 0) AS total_peserta,
// 			COALESCE(total_absen, 0) AS total_absen,
// 			COALESCE(total_male, 0) AS total_male,
// 			COALESCE(total_female, 0) AS total_female,
// 			COALESCE((total_absen / NULLIF(total_peserta, 0) * 100), 0) AS persen_hadir
// 		FROM (
// 			SELECT
// 				(SELECT COUNT(*)
// 				 FROM peserta
// 				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta
// 				 WHERE id_event = ?) AS total_peserta,

// 				(SELECT COUNT(DISTINCT user_id)
// 				 FROM absensi
// 				 WHERE event_id = ?
// 				 AND DATE(CONVERT_TZ(created_at, '+00:00', '+07:00')) = DATE(CONVERT_TZ(NOW(), '+00:00', '+07:00'))) AS total_absen,

// 				(SELECT COUNT(*)
// 				 FROM peserta
// 				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta
// 				 WHERE id_event = ? AND gender = 'male') AS total_male,

// 				(SELECT COUNT(*)
// 				 FROM peserta
// 				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta
// 				 WHERE id_event = ? AND gender = 'female') AS total_female
// 		) AS stats;
// 	`

// 	err = database.DB.QueryRow(query, eventID, eventID, eventID, eventID).Scan(&totalPeserta, &totalAbsen, &totalMale, &totalFemale, &persenHadir)
// 	if err != nil {
// 		log.Println("Error fetching event statistics:", err)
// 		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch event statistics"})
// 	}

// 	masjidStats := []map[string]interface{}{}
// 	rows, err := database.DB.Query(`
// 			SELECT
// 					m.id AS masjid_id,
// 					m.nama AS masjid_nama,
// 					m.alamat,
// 					regional.nama as nama_regional,
// 					COALESCE(COUNT(DISTINCT CASE WHEN peserta.gender = 'male' THEN absensi.user_id END), 0) AS male_count,
// 					COALESCE(COUNT(DISTINCT CASE WHEN peserta.gender = 'female' THEN absensi.user_id END), 0) AS female_count,
// 					COALESCE(COUNT(DISTINCT absensi.user_id), 0) AS total_count
// 			FROM masjid m
// 			LEFT JOIN regional ON regional.id = m.regional_id
// 			LEFT JOIN setting on setting.id_masjid = m.id
// 			LEFT JOIN petugas p ON p.id_masjid = m.id
// 			LEFT JOIN absensi ON p.id_user = absensi.mesin_id
// 					AND absensi.event_id = ?
// 					AND DATE(CONVERT_TZ(absensi.created_at, '+00:00', '+07:00')) = DATE(CONVERT_TZ(NOW(), '+00:00', '+07:00'))
// 			LEFT JOIN peserta ON absensi.user_id = peserta.id
// 			where setting.id_event = ?
// 			GROUP BY m.id, m.nama
// 			ORDER BY total_count DESC
// 	`, eventID, eventID)

// 	if err != nil {
// 		log.Println("Error fetching masjid attendance statistics:", err)
// 		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch masjid attendance statistics"})
// 	}
// 	defer rows.Close()

// 	// Menyimpan hasil query masjid
// 	for rows.Next() {
// 		var masjidID int
// 		var masjidNama string
// 		var masjidAlamat string
// 		var masjidRegional string
// 		var maleCount, femaleCount, totalCount int

// 		if err := rows.Scan(&masjidID, &masjidNama, &masjidAlamat, &masjidRegional, &maleCount, &femaleCount, &totalCount); err != nil {
// 			log.Println("Error scanning masjid row:", err)
// 			continue
// 		}

// 		masjidStats = append(masjidStats, map[string]interface{}{
// 			"masjid_id":       masjidID,
// 			"masjid_nama":     masjidNama,
// 			"masjid_alamat":   masjidAlamat,
// 			"masjid_regional": masjidRegional,
// 			"male_count":      maleCount,
// 			"female_count":    femaleCount,
// 			"total_count":     totalCount,
// 		})
// 	}

// 	// Return response JSON
// 	return c.JSON(fiber.Map{
// 		"event_id":      eventID,
// 		"total_peserta": totalPeserta,
// 		"total_absen":   totalAbsen,
// 		"total_male":    totalMale,
// 		"total_female":  totalFemale,
// 		"persen_hadir":  persenHadir,
// 		"masjid_stats":  masjidStats, // Data untuk bar chart
// 	})
// }

func GetEventStatistics(c *fiber.Ctx) error {
	// Ambil event_id dari query parameter
	eventIDStr := c.Query("event_id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil || eventID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "event_id is required"})
	}

	// Menentukan rentang waktu berdasarkan event_id
	var timeCondition string
	if eventID == 2 {
		// Rentang waktu dari jam 19:00 hari ini sampai 04:50 esok hari
		timeCondition = `
			(
				CONVERT_TZ(absensi.created_at, '+00:00', '+07:00') 
				BETWEEN CONCAT(CURDATE(), ' 19:00:00') 
				AND CONCAT(DATE_ADD(CURDATE(), INTERVAL 1 DAY), ' 06:00:00')
			)
		`
	} else {
		// Rentang waktu berdasarkan tanggal normal
		timeCondition = `
			DATE(CONVERT_TZ(created_at, '+00:00', '+07:00')) = DATE(CONVERT_TZ(NOW(), '+00:00', '+07:00'))
		`
	}

	// Query untuk mengambil statistik utama
	query := fmt.Sprintf(`
		SELECT
			COALESCE(total_peserta, 0) AS total_peserta,
			COALESCE(total_absen, 0) AS total_absen,
			COALESCE(total_male, 0) AS total_male,
			COALESCE(total_female, 0) AS total_female,
			COALESCE((total_absen / NULLIF(total_peserta, 0) * 100), 0) AS persen_hadir
		FROM (
			SELECT
				(SELECT COUNT(*) FROM peserta
				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta
				 WHERE id_event = ?) AS total_peserta,

				(SELECT COUNT(DISTINCT user_id) FROM absensi
				 WHERE event_id = ? AND %s) AS total_absen,

				(SELECT COUNT(*) FROM peserta
				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta
				 WHERE id_event = ? AND gender = 'male') AS total_male,

				(SELECT COUNT(*) FROM peserta
				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta
				 WHERE id_event = ? AND gender = 'female') AS total_female
		) AS stats;
	`, timeCondition)

	var totalPeserta, totalAbsen, totalMale, totalFemale int
	var persenHadir float64

	err = database.DB.QueryRow(query, eventID, eventID, eventID, eventID).Scan(&totalPeserta, &totalAbsen, &totalMale, &totalFemale, &persenHadir)
	if err != nil {
		log.Println("Error fetching event statistics:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch event statistics"})
	}

	masjidStats := []map[string]interface{}{}
	masjidQuery := fmt.Sprintf(`
		SELECT
			m.id AS masjid_id,
			m.nama AS masjid_nama,
			m.alamat,
			regional.nama as nama_regional,
			COALESCE(COUNT(DISTINCT CASE WHEN peserta.gender = 'male' THEN absensi.user_id END), 0) AS male_count,
			COALESCE(COUNT(DISTINCT CASE WHEN peserta.gender = 'female' THEN absensi.user_id END), 0) AS female_count,
			COALESCE(COUNT(DISTINCT absensi.user_id), 0) AS total_count
		FROM masjid m
		LEFT JOIN regional ON regional.id = m.regional_id
		LEFT JOIN setting ON setting.id_masjid = m.id
		LEFT JOIN petugas p ON p.id_masjid = m.id
		LEFT JOIN absensi ON p.id_user = absensi.mesin_id
			AND absensi.event_id = ?
			AND (
				CONVERT_TZ(absensi.created_at, '+00:00', '+07:00') 
				BETWEEN CONCAT(CURDATE(), ' 19:00:00') 
				AND CONCAT(DATE_ADD(CURDATE(), INTERVAL 1 DAY), ' 06:00:00')
			)
		LEFT JOIN peserta ON absensi.user_id = peserta.id
		WHERE setting.id_event = ?
		GROUP BY m.id, m.nama, m.alamat, regional.nama
		ORDER BY total_count DESC;
	`)

	rows, err := database.DB.Query(masjidQuery, eventID, eventID)
	if err != nil {
		log.Println("Error fetching masjid attendance statistics:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch masjid attendance statistics"})
	}
	defer rows.Close()

	for rows.Next() {
		var masjidID int
		var masjidNama, masjidAlamat, masjidRegional string
		var maleCount, femaleCount, totalCount int

		if err := rows.Scan(&masjidID, &masjidNama, &masjidAlamat, &masjidRegional, &maleCount, &femaleCount, &totalCount); err != nil {
			log.Println("Error scanning masjid row:", err)
			continue
		}

		masjidStats = append(masjidStats, map[string]interface{}{
			"masjid_id":       masjidID,
			"masjid_nama":     masjidNama,
			"masjid_alamat":   masjidAlamat,
			"masjid_regional": masjidRegional,
			"male_count":      maleCount,
			"female_count":    femaleCount,
			"total_count":     totalCount,
		})
	}

	return c.JSON(fiber.Map{
		"event_id":      eventID,
		"total_peserta": totalPeserta,
		"total_absen":   totalAbsen,
		"total_male":    totalMale,
		"total_female":  totalFemale,
		"persen_hadir":  persenHadir,
		"masjid_stats":  masjidStats,
	})
}
