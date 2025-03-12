package controllers

import (
	"net/http"
	"shollu/database"

	"github.com/gofiber/fiber/v2"
)

// Struct untuk response masjid
type Masjid struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Handler untuk mendapatkan daftar masjid
func GetMasjidList(c *fiber.Ctx) error {
	rows, err := database.DB.Query("SELECT id, nama FROM masjid")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch masjid"})
	}
	defer rows.Close()

	var masjids []Masjid

	// Iterasi hasil query
	for rows.Next() {
		var masjid Masjid
		if err := rows.Scan(&masjid.ID, &masjid.Name); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Error reading data"})
		}
		masjids = append(masjids, masjid)
	}

	// Cek jika data kosong
	if len(masjids) == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"message": "No masjid found"})
	}

	return c.JSON(fiber.Map{
		"message": "Success",
		"data":    masjids,
	})
}
