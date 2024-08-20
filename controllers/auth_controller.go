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

func Logout(c *fiber.Ctx) error {
	db := database.DBConn
	tokenString := c.Get("Authorization")

	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Authorization header missing",
		})
	}

	tokenString = tokenString[len("Bearer "):]

	// parse token เพื่ออ่าน token และดึง ACCESS_SECRET_KEY จาก env มาตรวจสอบว่า token ถูกต้องไหม
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("ACCESS_SECRET_KEY")), nil
	})

	// ตรวจสอบ token ที่ถูก parse ว่าถูกต้องหรือ (หมดอายุหรือถูกดัดแปลงไหม)
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired token")
	}

	// ดึง cliams จาก JWT
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid token claims.")
	}

	userID := uint(claims["UserID"].(float64))

	if err := db.Delete(&m.Session{}, "user_id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to log out.",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logged out successfully.",
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

	// ตั้งค่า originalTime และบันทึกลงใน Session
	session := m.Session{}
	if err := db.Where("user_id = ?", user.ID).First(&session).Error; err != nil {
		// ถ้าไม่มี session --> สร้าง session ใหม่
		session = m.Session{
			UserID:     user.ID,
			LastActive: time.Now(),
		}
		if err := db.Create(&session).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error creating session.")
		}
	} else {
		// ถ้ามี session อยู่แล้ว --> อัปเดต LastActive
		session.LastActive = time.Now()
		if err := db.Save(&session).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error updating session.")
		}
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

	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		// ตรวจสอบ signing method (for security)
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

func Approve(c *fiber.Ctx) error {
	db := database.DBConn
	var user m.User

	// รับค่า ID ของ user ที่ต้องการจะ approve
	inputID := c.FormValue("UserID")

	// ตรวจสอบ user ว่ามีอยู่ไหมจาก inputID
	if err := db.Where("id = ?", inputID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("User not found.")
	}

	// ตรวจสอบก่อนว่า user คนนี้ได้รับการ approve หรือยัง ถ้ายังก็ approve ให้กับ user คนนั้น
	if !user.Approve {
		user.Approve = true

		if err := db.Save(&user).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to approve user.")
		}
	} else {
		return c.Status(fiber.StatusBadRequest).SendString("This user has already been approved.")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": user.FirstName + " has been approved.",
	})
}

func GetUsers(c *fiber.Ctx) error {
	db := database.DBConn
	var users []m.User

	db.Find(&users)
	return c.Status(200).JSON(fiber.Map{
		"data": &users,
	})
}
