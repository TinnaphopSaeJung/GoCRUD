package controllers

import (
	"go-fiber-test/auth"
	"go-fiber-test/database"
	m "go-fiber-test/models"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *fiber.Ctx) error {
	db := database.DBConn
	var user m.User

	if err := c.BodyParser(&user); err != nil {
		return c.Status(503).SendString(err.Error())
	}

	// ตรวจสอบว่ามี User นี้อยู่แล้วไหม
	var existingUser m.User
	if err := db.Where("Username = ?", user.Username).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).SendString("User Already Exists.")
	}

	// Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error Hashing Password.")
	}
	user.Password = string(hashedPassword)

	// สร้างและบันทึกข้อมูลผู้ใช้ใหม่
	if err := db.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error Creating User.")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":    user,
		"message": "Register Success!!!",
	})
}

func Login(c *fiber.Ctx) error {
	db := database.DBConn
	var user m.User
	var input m.User

	if err := c.BodyParser(&input); err != nil {
		return c.Status(503).SendString(err.Error())
	}

	// ตรวจสอบว่า User นี้มีอยู่ระบบหรือไม่
	if err := db.Where("Username = ?", input.Username).First(&user).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid login, please try again.")
	}

	// ตรวจสอบว่า User นี้ได้รับการ Approve แล้วหรือยัง
	if !user.Approve {
		return c.Status(fiber.StatusBadRequest).SendString("This account has not been approved yet.")
	}

	// ตรวจสอบ Password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid login, please try again.")
	}

	// สร้าง Access Token
	accessSecretKey := os.Getenv("ACCESS_SECRET_KEY")
	accessToken, err := auth.GenerateToken(user, time.Minute*15, accessSecretKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error creating access token.")
	}

	// สร้าง Refresh Token
	refreshSecretKey := os.Getenv("REFRESH_SECRET_KEY")
	refreshToken, err := auth.GenerateToken(user, time.Hour*24, refreshSecretKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error creating refresh token.")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":      "Hello " + user.Username + ", You logged in Successfully.",
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"userId":       user.ID,
	})
}

func RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.FormValue("refreshToken")

	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("Refresh token required.")
	}

	// Verify the token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token signing method.")
		}
		return []byte(os.Getenv("REFRESH_SECRET_KEY")), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid refresh token.")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["UserID"] == nil {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid token claims.")
	}

	userID := claims["UserID"]

	// Retrieve user from the database
	var user m.User
	db := database.DBConn
	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("User not found.")
	}

	// สร้าง Access Token ใหม่
	accessSecretKey := os.Getenv("ACCESS_SECRET_KEY")
	newAccessToken, err := auth.GenerateToken(user, time.Minute*15, accessSecretKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error creating new access token.")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"accessToken": newAccessToken,
	})
}
