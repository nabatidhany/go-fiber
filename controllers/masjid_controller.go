package controllers

import (
	"net/http"
	"shollu/database"

	"github.com/gofiber/fiber/v2"
)

// Struct untuk response masjid
type Masjid struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Alamat string `json:"alamat"`
}

// Struct untuk response masjid
type MasjidGet struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Alamat string `json:"alamat"`
	Foto   string `json:"foto"`
}

// Handler untuk mendapatkan daftar masjid
func GetMasjidList(c *fiber.Ctx) error {
	idEvent := c.Params("id_event")
	rows, err := database.DB.Query("SELECT masjid.id, masjid.nama, masjid.alamat FROM masjid left join setting on masjid.id = setting.id_masjid where setting.id_event = ?", idEvent)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch masjid"})
	}
	defer rows.Close()

	var masjids []Masjid

	// Iterasi hasil query
	for rows.Next() {
		var masjid Masjid
		if err := rows.Scan(&masjid.ID, &masjid.Name, &masjid.Alamat); err != nil {
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

func GetMasjidByID(c *fiber.Ctx) error {
	idMasjid := c.Params("id_masjid")
	var masjid MasjidGet

	err := database.DB.QueryRow("SELECT id, nama, alamat, foto FROM masjid WHERE id = ?", idMasjid).Scan(&masjid.ID, &masjid.Name, &masjid.Alamat, &masjid.Foto)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Masjid not found"})
	}

	return c.JSON(fiber.Map{
		"message": "Success",
		"data":    masjid,
	})
}
