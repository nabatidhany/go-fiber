package controllers

import (
	"log"
	"shollu/database"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func GetEventStatistics(c *fiber.Ctx) error {
	// Ambil event_id dari query parameter
	eventIDStr := c.Query("event_id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil || eventID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "event_id is required"})
	}

	// Variabel untuk menyimpan hasil query
	var totalPeserta, totalAbsen, totalMale, totalFemale int
	var persenHadir float64

	// Query untuk mengambil statistik utama
	query := `
		SELECT 
			COALESCE(total_peserta, 0) AS total_peserta, 
			COALESCE(total_absen, 0) AS total_absen,
			COALESCE(total_male, 0) AS total_male,
			COALESCE(total_female, 0) AS total_female,
			COALESCE((total_absen / NULLIF(total_peserta, 0) * 100), 0) AS persen_hadir
		FROM (
			SELECT 
				(SELECT COUNT(*) 
				 FROM peserta 
				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta 
				 WHERE id_event = ?) AS total_peserta,

				(SELECT COUNT(DISTINCT user_id) 
				 FROM absensi 
				 WHERE event_id = ? 
				 AND DATE(created_at) = DATE(NOW())) AS total_absen,

				(SELECT COUNT(*) 
				 FROM peserta 
				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta 
				 WHERE id_event = ? AND gender = 'male') AS total_male,

				(SELECT COUNT(*) 
				 FROM peserta 
				 LEFT JOIN detail_peserta ON peserta.id = detail_peserta.id_peserta 
				 WHERE id_event = ? AND gender = 'female') AS total_female
		) AS stats;
	`

	err = database.DB.QueryRow(query, eventID, eventID, eventID, eventID).Scan(&totalPeserta, &totalAbsen, &totalMale, &totalFemale, &persenHadir)
	if err != nil {
		log.Println("Error fetching event statistics:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch event statistics"})
	}

	// Query untuk mendapatkan jumlah kehadiran pria & wanita di masing-masing masjid
	// masjidStats := []map[string]interface{}{}
	// rows, err := database.DB.Query(`
	// 	SELECT
	// 			m.id AS masjid_id,
	// 			m.nama AS masjid_nama,
	// 			COALESCE(male_count, 0) AS male_count,
	// 			COALESCE(female_count, 0) AS female_count
	// 	FROM masjid m
	// 	LEFT JOIN (
	// 			SELECT
	// 					p.id_masjid,
	// 					COUNT(DISTINCT CASE WHEN peserta.gender = 'male' THEN absensi.user_id END) AS male_count,
	// 					COUNT(DISTINCT CASE WHEN peserta.gender = 'female' THEN absensi.user_id END) AS female_count
	// 			FROM absensi
	// 			JOIN peserta ON absensi.user_id = peserta.id
	// 			JOIN petugas p ON absensi.mesin_id = p.id_user
	// 			WHERE absensi.event_id = ?
	// 			AND DATE(absensi.created_at) = DATE(NOW())
	// 			GROUP BY p.id_masjid
	// 	) AS absensi_stats ON m.id = absensi_stats.id_masjid
	// 	WHERE m.id IN (
	// 			SELECT DISTINCT petugas.id_masjid
	// 			FROM petugas
	// 			WHERE petugas.id_user IN (
	// 					SELECT DISTINCT mesin_id FROM absensi WHERE event_id = ?
	// 			)
	// 	)
	// `, eventID, eventID)

	// if err != nil {
	// 	log.Println("Error fetching masjid attendance statistics:", err)
	// 	return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch masjid attendance statistics"})
	// }
	// defer rows.Close()

	// // Menyimpan hasil query masjid
	// for rows.Next() {
	// 	var masjidID int
	// 	var masjidNama string
	// 	var maleCount, femaleCount int

	// 	if err := rows.Scan(&masjidID, &masjidNama, &maleCount, &femaleCount); err != nil {
	// 		log.Println("Error scanning masjid row:", err)
	// 		continue
	// 	}

	// 	masjidStats = append(masjidStats, map[string]interface{}{
	// 		"masjid_id":    masjidID,
	// 		"masjid_nama":  masjidNama,
	// 		"male_count":   maleCount,
	// 		"female_count": femaleCount,
	// 	})
	// }

	masjidStats := []map[string]interface{}{}
	rows, err := database.DB.Query(`
			SELECT 
					m.id AS masjid_id, 
					m.nama AS masjid_nama,
					COALESCE(COUNT(DISTINCT CASE WHEN peserta.gender = 'male' THEN absensi.user_id END), 0) AS male_count,
					COALESCE(COUNT(DISTINCT CASE WHEN peserta.gender = 'female' THEN absensi.user_id END), 0) AS female_count,
					COALESCE(COUNT(DISTINCT absensi.user_id), 0) AS total_count
			FROM masjid m
			LEFT JOIN petugas p ON p.id_masjid = m.id
			LEFT JOIN absensi ON p.id_user = absensi.mesin_id 
					AND absensi.event_id = ? 
					AND DATE(absensi.created_at) = DATE(NOW())
			LEFT JOIN peserta ON absensi.user_id = peserta.id
			GROUP BY m.id, m.nama
			ORDER BY total_count DESC
	`, eventID)

	if err != nil {
		log.Println("Error fetching masjid attendance statistics:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch masjid attendance statistics"})
	}
	defer rows.Close()

	// Menyimpan hasil query masjid
	for rows.Next() {
		var masjidID int
		var masjidNama string
		var maleCount, femaleCount, totalCount int

		if err := rows.Scan(&masjidID, &masjidNama, &maleCount, &femaleCount, &totalCount); err != nil {
			log.Println("Error scanning masjid row:", err)
			continue
		}

		masjidStats = append(masjidStats, map[string]interface{}{
			"masjid_id":    masjidID,
			"masjid_nama":  masjidNama,
			"male_count":   maleCount,
			"female_count": femaleCount,
			"total_count":  totalCount,
		})
	}

	// Return response JSON
	return c.JSON(fiber.Map{
		"event_id":      eventID,
		"total_peserta": totalPeserta,
		"total_absen":   totalAbsen,
		"total_male":    totalMale,
		"total_female":  totalFemale,
		"persen_hadir":  persenHadir,
		"masjid_stats":  masjidStats, // Data untuk bar chart
	})
}
