package controllers

import (
	"fmt"
	"net/http"
	"shollu/database"
	"shollu/utils"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CreateCollectionRequest struct {
	Name        string  `json:"name" validate:"required"`
	SholatTrack []int   `json:"sholat_track" validate:"required,dive,number,min=1,max=5"` // e.g. [1,2,3]
	DateStart   string  `json:"date_start" validate:"required"`                           // Format: YYYY-MM-DD
	DateEnd     string  `json:"date_end" validate:"required"`                             // Format: YYYY-MM-DD
	MasjidIDs   []int   `json:"masjid_id" validate:"required,dive,number"`
	PesertaIDs  []int64 `json:"peserta_ids" validate:"required,dive,required"`
}

func slugify(name string) string {
	return utils.GenerateSlug(name) // anggap utils ini sudah ada, atau bisa pakai strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

func CreateCollection(c *fiber.Ctx) error {
	var req CreateCollectionRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Validasi manual tanggal
	dateStart, err := time.Parse("2006-01-02", req.DateStart)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start date format"})
	}

	dateEnd, err := time.Parse("2006-01-02", req.DateEnd)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end date format"})
	}

	baseSlug := slugify(req.Name)
	slug, err := generateUniqueSlug(baseSlug)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate unique slug"})
	}
	sholatTrackStr := ""
	for i, v := range req.SholatTrack {
		if i > 0 {
			sholatTrackStr += ","
		}
		sholatTrackStr += fmt.Sprint(v)
	}
	trackingCode := sholatTrackStr
	now := time.Now()

	// Gabungkan masjid_id menjadi string dipisah koma
	masjidIDStr := ""
	if len(req.MasjidIDs) == 1 && req.MasjidIDs[0] == 0 {
		masjidIDStr = "all"
	} else {
		for i, id := range req.MasjidIDs {
			if i > 0 {
				masjidIDStr += ","
			}
			masjidIDStr += fmt.Sprint(id)
		}
	}

	// Insert ke tabel `collections`
	result, err := database.DB.Exec(`
		INSERT INTO collections (create_time, name, slug, tracking_code, date_start, date_end, masjid_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		now, req.Name, slug, trackingCode, dateStart, dateEnd, masjidIDStr)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create collection"})
	}

	collectionID, err := result.LastInsertId()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get collection ID"})
	}

	// Insert peserta ke collection_items
	for _, idPeserta := range req.PesertaIDs {
		_, err := database.DB.Exec(`
			INSERT INTO collection_items (create_time, collection_id, collection_slug, id_peserta)
			VALUES (?, ?, ?, ?)`, now, collectionID, slug, idPeserta)

		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to insert collection item for peserta ID %d", idPeserta),
			})
		}
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message":         "Collection created successfully",
		"collection_id":   collectionID,
		"tracking_code":   trackingCode,
		"slug":            slug,
		"sholat_tracking": req.SholatTrack,
	})
}

func generateUniqueSlug(baseSlug string) (string, error) {
	slug := baseSlug
	suffix := ""
	attempt := 0

	for {
		var count int
		err := database.DB.QueryRow("SELECT COUNT(*) FROM collections WHERE slug = ?", slug+suffix).Scan(&count)
		if err != nil {
			return "", err
		}

		if count == 0 {
			break
		}

		// Tambahkan random suffix jika sudah ada
		attempt++
		suffix = "-" + utils.RandomSuffix(4)
		slug = baseSlug + suffix

		if attempt > 10 {
			return "", fmt.Errorf("failed to generate unique slug")
		}
	}

	return slug + suffix, nil
}

func ViewCollection(c *fiber.Ctx) error {
	slug := c.Params("slug")

	// Ambil data collection
	var collection struct {
		ID          int64
		Name        string
		Slug        string
		DateStart   string
		DateEnd     string
		MasjidID    string // sudah VARCHAR, bisa "1,2,3" atau "all"
		SholatTrack string
	}
	err := database.DB.QueryRow(`
        SELECT id, name, slug, date_start, date_end, masjid_id, tracking_code 
        FROM collections 
        WHERE slug = ?`, slug).Scan(
		&collection.ID, &collection.Name, &collection.Slug,
		&collection.DateStart, &collection.DateEnd, &collection.MasjidID, &collection.SholatTrack,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Collection not found"})
	}

	// Map kode sholat ke nama tag
	sholatMap := map[string]string{
		"1": "subuh",
		"2": "dzuhur",
		"3": "ashar",
		"4": "maghrib",
		"5": "isya",
	}

	var sholatTags []string
	for _, code := range strings.Split(collection.SholatTrack, ",") {
		if tag, ok := sholatMap[code]; ok {
			sholatTags = append(sholatTags, tag)
		}
	}

	// Ambil peserta: id dan fullname
	pesertaRows, err := database.DB.Query(`
        SELECT p.id, p.fullname 
        FROM collection_items ci
        JOIN peserta p ON ci.id_peserta = p.id
        WHERE ci.collection_id = ?`, collection.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get peserta"})
	}
	defer pesertaRows.Close()

	pesertaMap := make(map[int]string)
	for pesertaRows.Next() {
		var id int
		var fullname string
		pesertaRows.Scan(&id, &fullname)
		pesertaMap[id] = fullname
	}
	if len(pesertaMap) == 0 {
		return c.JSON(fiber.Map{"message": "No peserta found"})
	}

	// Tanggal dari query
	dateFromStr := c.Query("date_from", time.Now().Format("2006-01-02"))
	dateToStr := c.Query("date_to", dateFromStr)

	dateFrom, err := time.Parse("2006-01-02", dateFromStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date_from format"})
	}
	dateTo, err := time.Parse("2006-01-02", dateToStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date_to format"})
	}

	var dates []string
	for d := dateFrom; !d.After(dateTo); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}

	// Build peserta ID list
	var pesertaIDs []string
	for id := range pesertaMap {
		pesertaIDs = append(pesertaIDs, fmt.Sprintf("%d", id))
	}
	inPeserta := strings.Join(pesertaIDs, ",")
	inTags := "'" + strings.Join(sholatTags, "','") + "'"

	// Build query absensi
	var absenQuery string
	if collection.MasjidID == "all" {
		absenQuery = fmt.Sprintf(`
			SELECT a.user_id, DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) as tanggal, a.tag
			FROM absensi a
			JOIN petugas p ON a.mesin_id = p.id_user
			WHERE a.tag IN (%s)
			  AND a.user_id IN (%s)
			  AND DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) BETWEEN '%s' AND '%s'
		`, inTags, inPeserta, dateFromStr, dateToStr)
	} else {
		masjidIDs := strings.Split(collection.MasjidID, ",")
		for i := range masjidIDs {
			masjidIDs[i] = strings.TrimSpace(masjidIDs[i])
		}
		inMasjid := strings.Join(masjidIDs, ",")

		absenQuery = fmt.Sprintf(`
			SELECT a.user_id, DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) as tanggal, a.tag
			FROM absensi a
			JOIN petugas p ON a.mesin_id = p.id_user
			WHERE p.id_masjid IN (%s)
			  AND a.tag IN (%s)
			  AND a.user_id IN (%s)
			  AND DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) BETWEEN '%s' AND '%s'
		`, inMasjid, inTags, inPeserta, dateFromStr, dateToStr)
	}

	// Jalankan query absensi
	absenRows, err := database.DB.Query(absenQuery)
	if err != nil {
		fmt.Println("Query Error:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get absensi"})
	}
	defer absenRows.Close()

	// Mapping absensi
	absensiMap := make(map[int]map[string]map[string]bool)
	for absenRows.Next() {
		var userID int
		var tanggalRaw time.Time
		var tag string
		absenRows.Scan(&userID, &tanggalRaw, &tag)

		tanggal := tanggalRaw.Format("2006-01-02")
		if _, ok := absensiMap[userID]; !ok {
			absensiMap[userID] = make(map[string]map[string]bool)
		}
		if _, ok := absensiMap[userID][tanggal]; !ok {
			absensiMap[userID][tanggal] = make(map[string]bool)
		}
		absensiMap[userID][tanggal][tag] = true
	}

	// Bangun response
	// var result []map[string]interface{}
	// for userID, fullname := range pesertaMap {
	// 	userAbsen := make(map[string]map[string]string)
	// 	for _, date := range dates {
	// 		userAbsen[date] = make(map[string]string)
	// 		for _, tag := range sholatTags {
	// 			if absensiMap[userID][date][tag] {
	// 				userAbsen[date][tag] = "Y"
	// 			} else {
	// 				userAbsen[date][tag] = "N"
	// 			}
	// 		}
	// 	}
	// 	result = append(result, map[string]interface{}{
	// 		"fullname": fullname,
	// 		"absen":    userAbsen,
	// 	})
	// }

	// Bangun response + hitung total 'Y' tiap peserta
	type pesertaData struct {
		Fullname string
		Absen    map[string]map[string]string
		TotalY   int
	}

	var result []pesertaData

	for userID, fullname := range pesertaMap {
		userAbsen := make(map[string]map[string]string)
		totalY := 0

		for _, date := range dates {
			userAbsen[date] = make(map[string]string)
			for _, tag := range sholatTags {
				if absensiMap[userID][date][tag] {
					userAbsen[date][tag] = "Y"
					totalY++
				} else {
					userAbsen[date][tag] = "N"
				}
			}
		}

		result = append(result, pesertaData{
			Fullname: fullname,
			Absen:    userAbsen,
			TotalY:   totalY,
		})
	}

	// Urutkan dari totalY terbanyak ke terkecil
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalY > result[j].TotalY
	})

	// Ubah ke []map[string]interface{} untuk response JSON
	var responseData []map[string]interface{}
	for _, r := range result {
		responseData = append(responseData, map[string]interface{}{
			"fullname": r.Fullname,
			"absen":    r.Absen,
			"total":    r.TotalY,
		})
	}

	return c.JSON(fiber.Map{
		"sholat_tracked": sholatTags,
		"dates":          dates,
		"data":           responseData,
	})

}

// func GetCollectionsMeta(c *fiber.Ctx) error {
// 	search := c.Query("search", "") // opsional pencarian nama

// 	query := `
// 		SELECT id, name, slug, date_start, date_end
// 		FROM collections
// 	`
// 	var args []interface{}

// 	if search != "" {
// 		query += " WHERE name LIKE ?"
// 		args = append(args, "%"+search+"%")
// 	}

// 	rows, err := database.DB.Query(query, args...)
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
// 			"error": "Failed to retrieve collections",
// 		})
// 	}
// 	defer rows.Close()

// 	type CollectionMeta struct {
// 		ID        int64  `json:"id"`
// 		Name      string `json:"name"`
// 		Slug      string `json:"slug"`
// 		DateStart string `json:"date_start"`
// 		DateEnd   string `json:"date_end"`
// 	}

// 	var result []CollectionMeta
// 	for rows.Next() {
// 		var col CollectionMeta
// 		err := rows.Scan(&col.ID, &col.Name, &col.Slug, &col.DateStart, &col.DateEnd)
// 		if err != nil {
// 			continue // skip error rows
// 		}
// 		result = append(result, col)
// 	}

// 	return c.JSON(fiber.Map{
// 		"collections": result,
// 	})
// }

func GetCollectionsMeta(c *fiber.Ctx) error {
	rows, err := database.DB.Query(`
		SELECT id, name, slug, tracking_code, date_start, date_end, masjid_id
		FROM collections ORDER BY create_time DESC
	`)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch collections"})
	}
	defer rows.Close()

	sholatMap := map[string]string{
		"1": "subuh",
		"2": "dzuhur",
		"3": "ashar",
		"4": "maghrib",
		"5": "isya",
	}

	var collections []map[string]interface{}

	for rows.Next() {
		var id int
		var name, slug, trackingCode, dateStart, dateEnd, masjidID string
		if err := rows.Scan(&id, &name, &slug, &trackingCode, &dateStart, &dateEnd, &masjidID); err != nil {
			continue
		}

		// Get peserta IDs
		pesertaRows, err := database.DB.Query(`SELECT id_peserta FROM collection_items WHERE collection_id = ?`, id)
		if err != nil {
			continue
		}

		var pesertaIDs []string
		for pesertaRows.Next() {
			var pid int
			pesertaRows.Scan(&pid)
			pesertaIDs = append(pesertaIDs, fmt.Sprintf("%d", pid))
		}
		pesertaRows.Close()
		if len(pesertaIDs) == 0 {
			continue
		}

		inPeserta := strings.Join(pesertaIDs, ",")
		trackedSholat := strings.Split(trackingCode, ",")
		var sholatTags []string
		for _, code := range trackedSholat {
			if tag, ok := sholatMap[code]; ok {
				sholatTags = append(sholatTags, tag)
			}
		}
		inTags := "'" + strings.Join(sholatTags, "','") + "'"

		// Build absensi query
		var absenQuery string
		if masjidID == "all" {
			// absenQuery = fmt.Sprintf(`
			// 	SELECT a.tag, COUNT(*) as total
			// 	FROM absensi a
			// 	JOIN petugas p ON a.mesin_id = p.id_user
			// 	WHERE a.user_id IN (%s) AND a.tag IN (%s)
			// 	AND DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) BETWEEN '%s' AND '%s'
			// 	GROUP BY a.tag
			// `, inPeserta, inTags, dateStart, dateEnd)
			absenQuery = fmt.Sprintf(`
				SELECT tag, COUNT(*) as total FROM (
					SELECT DISTINCT a.user_id, a.tag, DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) as tanggal
					FROM absensi a
					JOIN petugas p ON a.mesin_id = p.id_user
					WHERE a.user_id IN (%s)
					AND a.tag IN (%s)
					AND DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) BETWEEN '%s' AND '%s'
				) as unique_daily_absen
				GROUP BY tag
			`, inPeserta, inTags, dateStart, dateEnd)

		} else {
			masjidIDs := strings.Split(masjidID, ",")
			for i := range masjidIDs {
				masjidIDs[i] = strings.TrimSpace(masjidIDs[i])
			}
			inMasjid := strings.Join(masjidIDs, ",")
			// absenQuery = fmt.Sprintf(`
			// 	SELECT a.tag, COUNT(*) as total
			// 	FROM absensi a
			// 	JOIN petugas p ON a.mesin_id = p.id_user
			// 	WHERE p.id_masjid IN (%s)
			// 	AND a.user_id IN (%s)
			// 	AND a.tag IN (%s)
			// 	AND DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) BETWEEN '%s' AND '%s'
			// 	GROUP BY a.tag
			// `, inMasjid, inPeserta, inTags, dateStart, dateEnd)
			absenQuery = fmt.Sprintf(`
				SELECT tag, COUNT(*) as total FROM (
					SELECT DISTINCT a.user_id, a.tag, DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) as tanggal
					FROM absensi a
					JOIN petugas p ON a.mesin_id = p.id_user
					WHERE p.id_masjid IN (%s)
					AND a.user_id IN (%s)
					AND a.tag IN (%s)
					AND DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) BETWEEN '%s' AND '%s'
				) as unique_daily_absen
				GROUP BY tag
			`, inMasjid, inPeserta, inTags, dateStart, dateEnd)

		}

		// Execute absensi summary query
		summaryRows, err := database.DB.Query(absenQuery)
		if err != nil {
			continue
		}

		summaries := make(map[string]int)
		for summaryRows.Next() {
			var tag string
			var total int
			summaryRows.Scan(&tag, &total)
			summaries[tag] = total
		}
		summaryRows.Close()

		collections = append(collections, fiber.Map{
			"id":         id,
			"name":       name,
			"slug":       slug,
			"start_date": dateStart,
			"end_date":   dateEnd,
			"summaries":  summaries,
		})
	}

	return c.JSON(fiber.Map{
		"collections": collections,
	})
}

func GetCollectionsMetaDetail(c *fiber.Ctx) error {
	slug := c.Params("slug")
	query := `
		SELECT id, name, slug, date_start, date_end, masjid_id
		FROM collections WHERE slug = ?
	`

	row := database.DB.QueryRow(query, slug)

	type CollectionMeta struct {
		ID          int64    `json:"id"`
		Name        string   `json:"name"`
		Slug        string   `json:"slug"`
		DateStart   string   `json:"date_start"`
		DateEnd     string   `json:"date_end"`
		MasjidID    string   `json:"masjid_id"`
		MasjidNames []string `json:"masjid_names"`
	}

	var result CollectionMeta
	err := row.Scan(&result.ID, &result.Name, &result.Slug, &result.DateStart, &result.DateEnd, &result.MasjidID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Collection not found",
		})
	}

	// Ambil nama masjid
	var masjidQuery string
	if result.MasjidID == "all" {
		masjidQuery = `SELECT nama FROM masjid`
	} else {
		ids := strings.Split(result.MasjidID, ",")
		for i := range ids {
			ids[i] = strings.TrimSpace(ids[i])
		}
		inClause := "'" + strings.Join(ids, "','") + "'"
		masjidQuery = fmt.Sprintf(`SELECT nama FROM masjid WHERE id IN (%s)`, inClause)
	}

	rows, err := database.DB.Query(masjidQuery)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get masjid names",
		})
	}
	defer rows.Close()

	for rows.Next() {
		var nama string
		rows.Scan(&nama)
		result.MasjidNames = append(result.MasjidNames, nama)
	}

	return c.JSON(fiber.Map{
		"collections": result,
	})
}

// func GetCollectionsMetaDetail(c *fiber.Ctx) error {
// 	slug := c.Params("slug")
// 	query := `
// 		SELECT id, name, slug, date_start, date_end, masjid_id
// 		FROM collections WHERE slug = ?
// 	`

// 	row := database.DB.QueryRow(query, slug)

// 	type CollectionMeta struct {
// 		ID        int64  `json:"id"`
// 		Name      string `json:"name"`
// 		Slug      string `json:"slug"`
// 		DateStart string `json:"date_start"`
// 		DateEnd   string `json:"date_end"`
// 		MasjidID  string `json:"masjid_id"`
// 	}

// 	var result CollectionMeta
// 	err := row.Scan(&result.ID, &result.Name, &result.Slug, &result.DateStart, &result.DateEnd, &result.MasjidID)
// 	if err != nil {
// 		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
// 			"error": "Collection not found",
// 		})
// 	}

// 	return c.JSON(fiber.Map{
// 		"collections": result,
// 	})
// }

// func ViewCollection(c *fiber.Ctx) error {
// 	slug := c.Params("slug")

// 	// Ambil data collection
// 	var collection struct {
// 		ID          int64
// 		Name        string
// 		Slug        string
// 		DateStart   string
// 		DateEnd     string
// 		MasjidID    string
// 		SholatTrack string
// 	}
// 	err := database.DB.QueryRow(`
//         SELECT id, name, slug, date_start, date_end, masjid_id, tracking_code
//         FROM collections
//         WHERE slug = ?`, slug).Scan(
// 		&collection.ID, &collection.Name, &collection.Slug,
// 		&collection.DateStart, &collection.DateEnd, &collection.MasjidID, &collection.SholatTrack,
// 	)
// 	if err != nil {
// 		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Collection not found"})
// 	}

// 	// Map kode sholat ke nama tag
// 	sholatMap := map[string]string{
// 		"1": "subuh",
// 		"2": "dzuhur",
// 		"3": "ashar",
// 		"4": "maghrib",
// 		"5": "isya",
// 	}

// 	// Ambil sholat tags dari collection
// 	var sholatTags []string
// 	for _, code := range strings.Split(collection.SholatTrack, ",") {
// 		if tag, ok := sholatMap[code]; ok {
// 			sholatTags = append(sholatTags, tag)
// 		}
// 	}

// 	// Ambil peserta: id dan fullname
// 	pesertaRows, err := database.DB.Query(`
//         SELECT p.id, p.fullname
//         FROM collection_items ci
//         JOIN peserta p ON ci.id_peserta = p.id
//         WHERE ci.collection_id = ?`, collection.ID)
// 	if err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get peserta"})
// 	}
// 	defer pesertaRows.Close()

// 	pesertaMap := make(map[int]string) // id -> fullname
// 	for pesertaRows.Next() {
// 		var id int
// 		var fullname string
// 		pesertaRows.Scan(&id, &fullname)
// 		pesertaMap[id] = fullname
// 	}

// 	if len(pesertaMap) == 0 {
// 		return c.JSON(fiber.Map{"message": "No peserta found"})
// 	}

// 	// Ambil tanggal dari parameter, default ke hari ini jika tidak ada
// 	dateFromStr := c.Query("date_from", time.Now().Format("2006-01-02"))
// 	dateToStr := c.Query("date_to", dateFromStr)

// 	// Parse tanggal
// 	dateFrom, err := time.Parse("2006-01-02", dateFromStr)
// 	if err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date_from format"})
// 	}
// 	dateTo, err := time.Parse("2006-01-02", dateToStr)
// 	if err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date_to format"})
// 	}

// 	// Generate list tanggal antara dateFrom dan dateTo
// 	var dates []string
// 	for d := dateFrom; !d.After(dateTo); d = d.AddDate(0, 0, 1) {
// 		dates = append(dates, d.Format("2006-01-02"))
// 	}

// 	// Ambil semua absensi dalam periode untuk semua peserta dan tag
// 	pesertaIDs := []string{}
// 	for id := range pesertaMap {
// 		pesertaIDs = append(pesertaIDs, fmt.Sprintf("%d", id))
// 	}
// 	inPeserta := strings.Join(pesertaIDs, ",")

// 	inTags := "'" + strings.Join(sholatTags, "','") + "'"

// 	// Cek dan format masjid_id agar aman untuk query IN
// 	masjidIDs := []string{}
// 	for _, id := range strings.Split(collection.MasjidID, ",") {
// 		id = strings.TrimSpace(id)
// 		if id != "" {
// 			masjidIDs = append(masjidIDs, id)
// 		}
// 	}
// 	inMasjid := strings.Join(masjidIDs, ",")

// 	// Query absensi
// 	absenQuery := fmt.Sprintf(`
//         SELECT a.user_id, DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) as tanggal, a.tag
//         FROM absensi a
//         JOIN petugas p ON a.mesin_id = p.id_user
//         WHERE p.id_masjid IN (%s)
//           AND a.tag IN (%s)
//           AND a.user_id IN (%s)
//           AND DATE(CONVERT_TZ(a.created_at, '+00:00', '+07:00')) BETWEEN '%s' AND '%s'
//     `, inMasjid, inTags, inPeserta, dateFromStr, dateToStr)

// 	absenRows, err := database.DB.Query(absenQuery)
// 	if err != nil {
// 		fmt.Println("Error executing query:", err)
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get absensi"})
// 	}
// 	defer absenRows.Close()

// 	// Map absensi: user_id -> tanggal -> tag -> true
// 	absensiMap := make(map[int]map[string]map[string]bool)
// 	for absenRows.Next() {
// 		var userID int
// 		var tanggalRaw time.Time
// 		var tag string
// 		absenRows.Scan(&userID, &tanggalRaw, &tag)

// 		tanggal := tanggalRaw.Format("2006-01-02")

// 		if _, ok := absensiMap[userID]; !ok {
// 			absensiMap[userID] = make(map[string]map[string]bool)
// 		}
// 		if _, ok := absensiMap[userID][tanggal]; !ok {
// 			absensiMap[userID][tanggal] = make(map[string]bool)
// 		}
// 		absensiMap[userID][tanggal][tag] = true
// 	}

// 	// Bangun output
// 	var result []map[string]interface{}
// 	for userID, fullname := range pesertaMap {
// 		userAbsen := make(map[string]map[string]string)

// 		for _, date := range dates {
// 			userAbsen[date] = make(map[string]string)
// 			for _, tag := range sholatTags {
// 				if absensiMap[userID][date][tag] {
// 					userAbsen[date][tag] = "Y"
// 				} else {
// 					userAbsen[date][tag] = "N"
// 				}
// 			}
// 		}

// 		result = append(result, map[string]interface{}{
// 			"fullname": fullname,
// 			"absen":    userAbsen,
// 		})
// 	}

// 	return c.JSON(fiber.Map{
// 		"sholat_tracked": sholatTags,
// 		"dates":          dates,
// 		"data":           result,
// 	})
// }
