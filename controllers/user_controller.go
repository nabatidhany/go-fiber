package controllers

import (
	"database/sql"
	"shollu/database"
	"shollu/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

func GetProfile(c *fiber.Ctx) error {
	// Ambil data user dari token JWT
	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	userID := claims["id"].(float64) // JWT menyimpan angka sebagai float64

	// Ambil data user dari database
	var user models.User
	err := database.DB.QueryRow("SELECT id, name, email FROM users WHERE id = ?", int(userID)).
		Scan(&user.ID, &user.Name, &user.Email)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
	}

	// Kirim data user sebagai respons
	return c.JSON(user)
}
