package database

import (
	"database/sql"
	"fmt"
	"time"

	"shollu/config"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() {
	var err error

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		config.DBUser, config.DBPassword, config.DBHost, config.DBName)

	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	// Konfigurasi Database Pooling
	DB.SetMaxOpenConns(25)                 // Maksimum koneksi terbuka
	DB.SetMaxIdleConns(10)                 // Maksimum koneksi idle
	DB.SetConnMaxLifetime(5 * time.Minute) // Waktu hidup koneksi

	// Cek koneksi
	if err = DB.Ping(); err != nil {
		panic(err)
	}
}
