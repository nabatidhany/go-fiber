package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"shollu/database"
	"strconv"
	"time"

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

	// Ambil event_date dari query parameter, default ke hari ini jika tidak dikirim
	eventDate := c.Query("event_date")
	if eventDate == "" {
		eventDate = time.Now().Format("2006-01-02") // Format YYYY-MM-DD
	}

	// Menentukan rentang waktu berdasarkan event_id
	var timeCondition string
	if eventID == 2 {
		// Rentang waktu dari jam 19:00 event_date sampai 06:00 event_date +1
		timeCondition = `
			(
				CONVERT_TZ(absensi.created_at, '+00:00', '+07:00')
				BETWEEN CONCAT(?, ' 19:00:00')
				AND CONCAT(DATE_ADD(?, INTERVAL 1 DAY), ' 06:00:00')
			)
		`
	} else {
		// Rentang waktu berdasarkan tanggal normal
		timeCondition = `
			DATE(absensi.created_at) = DATE(?)
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

	if eventID == 2 {
		err = database.DB.QueryRow(query, eventID, eventID, eventDate, eventDate, eventID, eventID).Scan(&totalPeserta, &totalAbsen, &totalMale, &totalFemale, &persenHadir)
	} else {
		err = database.DB.QueryRow(query, eventID, eventID, eventDate, eventID, eventID).Scan(&totalPeserta, &totalAbsen, &totalMale, &totalFemale, &persenHadir)
	}

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
			AND %s
		LEFT JOIN peserta ON absensi.user_id = peserta.id
		WHERE setting.id_event = ?
		GROUP BY m.id, m.nama, m.alamat, regional.nama
		ORDER BY total_count DESC;
	`, timeCondition)

	var rows *sql.Rows
	if eventID == 2 {
		rows, err = database.DB.Query(masjidQuery, eventID, eventDate, eventDate, eventID)
	} else {
		rows, err = database.DB.Query(masjidQuery, eventID, eventDate, eventID)
	}

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
		"event_date":    eventDate,
		"total_peserta": totalPeserta,
		"total_absen":   totalAbsen,
		"total_male":    totalMale,
		"total_female":  totalFemale,
		"persen_hadir":  persenHadir,
		"masjid_stats":  masjidStats,
	})
}

// func GetEventStatistics(c *fiber.Ctx) error {
// 	eventIDStr := c.Query("event_id")
// 	eventDate := c.Query("event_date")

// 	eventID, err := strconv.Atoi(eventIDStr)
// 	if err != nil {
// 		return c.Status(400).JSON(fiber.Map{"error": "Invalid event_id"})
// 	}

// 	var masjidQuery string
// 	var rows *sql.Rows

// 	// Khusus untuk event_id == 3 (hitung tag unik per user)
// 	if eventID == 3 {
// 		masjidQuery = `
// 			SELECT
// 					subquery.masjid_id,
// 					subquery.masjid_nama,
// 					subquery.alamat,
// 					subquery.nama_regional,
// 					COALESCE(SUM(subquery.male_count), 0) AS male_count,
// 					COALESCE(SUM(subquery.female_count), 0) AS female_count,
// 					COALESCE(SUM(subquery.unique_tags), 0) AS total_count
// 			FROM (
// 					SELECT
// 							m.id as masjid_id,
// 							m.nama as masjid_nama,
// 							m.alamat,
// 							regional.nama as nama_regional,
// 							absensi.user_id,
// 							COUNT(DISTINCT absensi.tag) as unique_tags,
// 							MAX(CASE WHEN peserta.gender = 'male' THEN 1 ELSE 0 END) as male_count,
// 							MAX(CASE WHEN peserta.gender = 'female' THEN 1 ELSE 0 END) as female_count
// 					FROM masjid m
// 					LEFT JOIN regional ON regional.id = m.regional_id
// 					LEFT JOIN setting ON setting.id_masjid = m.id
// 					LEFT JOIN petugas p ON p.id_masjid = m.id
// 					LEFT JOIN absensi ON p.id_user = absensi.mesin_id
// 							AND absensi.event_id = ?
// 							AND DATE(CONVERT_TZ(absensi.created_at, '+00:00', '+07:00')) = DATE(?)
// 							AND absensi.tag IS NOT NULL
// 					LEFT JOIN peserta ON absensi.user_id = peserta.id
// 					WHERE setting.id_event = ?
// 					GROUP BY m.id, m.nama, m.alamat, regional.nama, absensi.user_id
// 			) AS subquery
// 			GROUP BY subquery.masjid_id, subquery.masjid_nama, subquery.alamat, subquery.nama_regional
// 			ORDER BY total_count DESC

// 		`

// 		rows, err = database.DB.Query(masjidQuery, eventID, eventDate, eventID)
// 		if err != nil {
// 			log.Println("Error fetching event 3 statistics:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch event 3 statistics"})
// 		}
// 	} else {
// 		// Default case untuk event selain 3
// 		timeCondition := "DATE(CONVERT_TZ(absensi.created_at, '+00:00', '+07:00')) = DATE(?)"
// 		if eventID == 2 {
// 			timeCondition = "DATE(CONVERT_TZ(absensi.created_at, '+00:00', '+07:00')) BETWEEN DATE(?) AND DATE(?)"
// 		}

// 		masjidQuery = fmt.Sprintf(`
// 			SELECT
// 				m.id AS masjid_id,
// 				m.nama AS masjid_nama,
// 				m.alamat,
// 				regional.nama as nama_regional,
// 				COALESCE(COUNT(DISTINCT CASE WHEN peserta.gender = 'male' THEN absensi.user_id END), 0) AS male_count,
// 				COALESCE(COUNT(DISTINCT CASE WHEN peserta.gender = 'female' THEN absensi.user_id END), 0) AS female_count,
// 				COALESCE(COUNT(DISTINCT absensi.user_id), 0) AS total_count
// 			FROM masjid m
// 			LEFT JOIN regional ON regional.id = m.regional_id
// 			LEFT JOIN setting ON setting.id_masjid = m.id
// 			LEFT JOIN petugas p ON p.id_masjid = m.id
// 			LEFT JOIN absensi ON p.id_user = absensi.mesin_id
// 				AND absensi.event_id = ?
// 				AND %s
// 			LEFT JOIN peserta ON absensi.user_id = peserta.id
// 			WHERE setting.id_event = ?
// 			GROUP BY m.id, m.nama, m.alamat, regional.nama
// 			ORDER BY total_count DESC;
// 		`, timeCondition)

// 		if eventID == 2 {
// 			rows, err = database.DB.Query(masjidQuery, eventID, eventDate, eventDate, eventID)
// 		} else {
// 			rows, err = database.DB.Query(masjidQuery, eventID, eventDate, eventID)
// 		}

// 		if err != nil {
// 			log.Println("Error fetching masjid statistics:", err)
// 			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch masjid statistics"})
// 		}
// 	}

// 	defer rows.Close()

// 	var masjidStats []fiber.Map

// 	for rows.Next() {
// 		var masjidID int
// 		var nama, alamat, regional string
// 		var maleCount, femaleCount, totalCount int

// 		err := rows.Scan(&masjidID, &nama, &alamat, &regional, &maleCount, &femaleCount, &totalCount)
// 		if err != nil {
// 			log.Println("Error scanning masjid stat:", err)
// 			continue
// 		}

// 		masjidStats = append(masjidStats, fiber.Map{
// 			"masjid_id":       masjidID,
// 			"masjid_nama":     nama,
// 			"masjid_alamat":   alamat,
// 			"masjid_regional": regional,
// 			"male_count":      maleCount,
// 			"female_count":    femaleCount,
// 			"total_count":     totalCount,
// 		})
// 	}

// 	return c.JSON(fiber.Map{
// 		"event_id":     eventID,
// 		"event_date":   eventDate,
// 		"masjid_stats": masjidStats,
// 	})
// }

func GetAttendanceStatistics(c *fiber.Ctx) error {
	startDate := "2025-03-20"
	endDate := "2025-03-29"
	eventID := 2

	query := `
		SELECT
			dates.date AS event_date,
			COALESCE(
				(
					(SELECT COUNT(DISTINCT user_id) FROM absensi
					 WHERE event_id = ? AND 
					 CONVERT_TZ(absensi.created_at, '+00:00', '+07:00') 
					 BETWEEN CONCAT(dates.date, ' 19:00:00') 
					 AND CONCAT(DATE_ADD(dates.date, INTERVAL 1 DAY), ' 06:00:00'))
					/
					NULLIF((SELECT COUNT(*) FROM peserta
					LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta
					WHERE detail_peserta.id_event = ?), 0) * 100
				), 0) AS persen_hadir,
			COALESCE((SELECT COUNT(*) FROM peserta
				LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta
				WHERE detail_peserta.id_event = ?), 0) AS total_peserta,
			COALESCE((SELECT COUNT(DISTINCT user_id) FROM absensi
				 WHERE event_id = ? AND 
				 CONVERT_TZ(absensi.created_at, '+00:00', '+07:00') 
				 BETWEEN CONCAT(dates.date, ' 19:00:00') 
				 AND CONCAT(DATE_ADD(dates.date, INTERVAL 1 DAY), ' 06:00:00')), 0) AS total_hadir
		FROM (
			SELECT DATE_ADD(?, INTERVAL seq DAY) AS date
			FROM (SELECT 0 AS seq UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4
			      UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9
			      UNION ALL SELECT 10 UNION ALL SELECT 11 UNION ALL SELECT 12 UNION ALL SELECT 13 UNION ALL SELECT 14
			      UNION ALL SELECT 15 UNION ALL SELECT 16 UNION ALL SELECT 17 UNION ALL SELECT 18 UNION ALL SELECT 19) AS seq
			WHERE DATE_ADD(?, INTERVAL seq DAY) <= ? COLLATE utf8mb4_unicode_ci
		) AS dates;
	`

	rows, err := database.DB.Query(query, eventID, eventID, eventID, eventID, startDate, startDate, endDate)
	if err != nil {
		log.Println("Error fetching attendance statistics:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch attendance statistics"})
	}
	defer rows.Close()

	attendanceStats := []map[string]interface{}{}

	for rows.Next() {
		var eventDate string
		var persenHadir float64
		var totalPeserta int
		var totalHadir int
		if err := rows.Scan(&eventDate, &persenHadir, &totalPeserta, &totalHadir); err != nil {
			log.Println("Error scanning row:", err)
			continue
		}
		attendanceStats = append(attendanceStats, map[string]interface{}{
			"date":          eventDate,
			"persen_hadir":  persenHadir,
			"total_peserta": totalPeserta,
			"total_hadir":   totalHadir,
		})
	}

	return c.JSON(attendanceStats)
}
