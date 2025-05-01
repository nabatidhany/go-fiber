package utils

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

func GenerateSlug(name string) string {
	// Ubah ke huruf kecil, ganti spasi, dan hapus karakter non-alfanumerik (selain -)
	slug := strings.ToLower(name)
	slug = strings.TrimSpace(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	reg := regexp.MustCompile("[^a-z0-9-]+")
	slug = reg.ReplaceAllString(slug, "")
	return slug
}

func RandomSuffix(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n] // contoh hasil: "a1c3"
}
