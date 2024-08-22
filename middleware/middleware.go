package middleware

import (
	"go-fiber-test/database"
	m "go-fiber-test/models"
	"go-fiber-test/session"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired(c *fiber.Ctx) error {
	// ดึง Authorization header
	tokenString := c.Get("Authorization")

	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("Authorization header missing.")
	}

	// ตัด Bearer ออกให้เหลือแค่ token
	tokenString = tokenString[len("Bearer "):]

	// parse token เพื่ออ่าน token และดึง ACCESS_SECRET_KEY จาก env มาตรวจสอบว่า token ถูกต้องไหม
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// ตรวจสอบ signing method (for security)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token signing method.")
		}
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

	// เซ็ต claims ลงใน context เพื่อใช้งานใน controller
	c.Locals("user", claims)

	// สร้างตัวแปรมาเก็บค่าเวลาเดิมก่อนที่จะส่ง request (ครั้งแรกที่ login จะเป็นค่าเวลาเป็นเวลาปัจจุบันก่อน)
	originalSession := m.Session{}
	db := database.DBConn
	userID := uint(claims["UserID"].(float64))
	if err := db.Where("user_id = ?", userID).First(&originalSession).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not retrieve session data.",
		})
	}

	// update LastActive เมื่อเริ่มส่ง request
	if err := session.UpdateSessionActivity(userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not update session activity.",
		})
	}

	// หลังจาก update ค่า LastActive ก็สร้างตัวแปรมาเก็บค่าเวลาตอนที่ส่ง request
	dbSession := m.Session{}
	if err := db.Where("user_id = ?", userID).First(&dbSession).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not retrieve session data.",
		})
	}

	// ตรวจสอบว่า session หมดเวลาแล้วหรือยัง
	sessionTimeout := time.Minute * 10
	lastActive := dbSession.LastActive
	originalTime := originalSession.LastActive
	timeDifference := lastActive.Sub(originalTime)

	if timeDifference > sessionTimeout {
		// ลบ session ออกจากฐานข้อมูล
		if err := db.Delete(&dbSession).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Could not delete expired session.",
			})
		}

		// แจ้งเตือนให้ผู้ใช้ login ใหม่
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Session expired, Please login again.",
		})
	}

	// update LastActive หลังจากตรวจสอบ session timeout
	dbSession.LastActive = time.Now()
	if err := db.Save(&dbSession).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not update session activity.",
		})
	}

	return c.Next()
}

func RoleRequired(requiredRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")

		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).SendString("Authorization header missing.")
		}

		tokenString = tokenString[len("Bearer "):]

		// parse token เพื่ออ่าน token และดึง ACCESS_SECRET_KEY จาก env มาตรวจสอบว่า token ถูกต้องไหม
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// ตรวจสอบ signing method (for security)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token signing method.")
			}
			return []byte(os.Getenv("ACCESS_SECRET_KEY")), nil
		})

		// ตรวจสอบ token ที่ถูก parse ว่าถูกต้องหรือ (หมดอายุหรือถูกดัดแปลงไหม)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired token")
		}

		// ดึง Claims จาก JWT
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid token claims.")
		}

		// ดึง Role จาก Claims และตรวจสอบว่าตรงกับ requiredRole หรือไม่
		userRole := claims["Role"].(string)
		if userRole != requiredRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Access denied for this role.",
			})
		}

		return c.Next()
	}
}
