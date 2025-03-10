package config

import (
	"os"

	"github.com/joho/godotenv"
)

var (
	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	JWTSecret  string
)

func LoadConfig() {
	godotenv.Load()
	DBUser = os.Getenv("DB_USER")
	DBPassword = os.Getenv("DB_PASSWORD")
	DBName = os.Getenv("DB_NAME")
	DBHost = os.Getenv("DB_HOST")
	JWTSecret = os.Getenv("JWT_SECRET")
}
