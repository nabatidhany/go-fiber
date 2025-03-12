package controllers

import (
	"database/sql"
	"net/http"
	"shollu/config"
	"shollu/database"
	"shollu/models"
	"shollu/utils"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var validate = validator.New()

func Register(c *fiber.Ctx) error {
	// Parse body JSON
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Validasi input
	err := validate.Struct(req)
	if err != nil {
		errors := utils.FormatValidationErrors(err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"errors": errors})
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	// Simpan ke database
	_, err = database.DB.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)", req.Username, req.Email, hashedPassword)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to register user"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "User registered successfully"})
}

func Login(c *fiber.Ctx) error {
	input := new(LoginInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
	}

	var user models.User
	err := database.DB.QueryRow("SELECT id, name, email, password FROM users WHERE email = ?", input.Email).
		Scan(&user.ID, &user.Name, &user.Email, &user.Password)
	if err == sql.ErrNoRows || !utils.CheckPassword(user.Password, input.Password) {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  user.ID,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	})
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Could not create token"})
	}

	return c.JSON(fiber.Map{"token": tokenString})
}
