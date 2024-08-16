package auth

import (
	m "go-fiber-test/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(user m.User, expiryTime time.Duration, secretKey string) (string, error) {
	claims := jwt.MapClaims{
		"Username": user.Username,
		"Role":     user.Role,
		"UserID":   user.ID,
		"exp":      time.Now().Add(expiryTime).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}
