package utils

import (
	"database/sql"
	"log"
	"net/http"
)

// isModerator kullanıcı ID'sine göre kullanıcının moderatör olup olmadığını kontrol eder

func IsModerator(userID int, db *sql.DB) (bool, error) {
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // Kullanıcı bulunamadı, moderatör değil
		}
		return false, err // Başka bir hata oluştu
	}
	return role == "moderator", nil
}

func IsAdmin(r *http.Request) bool {
	userID, err := GetUserIDFromCookie(r)
	if err != nil {
		return false
	}

	var role string
	err = Db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
	if err != nil {
		log.Println(err)
		return false
	}

	return role == "admin"
}