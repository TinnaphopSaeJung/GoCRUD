package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired(c *fiber.Ctx) error {
	// ดึง Authorization header
	tokenString := c.Get("Authorization")

	// ถ้าไม่พบ token ให้ส่งสถานะ Unauthorized กลับไป
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Authorization header missing",
		})
	}

	// ตรวจสอบว่า token เป็น Bearer token หรือไม่
	tokenString = tokenString[len("Bearer "):]

	// Parse และตรวจสอบ token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// ตรวจสอบว่า token ใช้ method ที่ถูกต้อง
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token signing method.")
		}
		// ใช้ Secret Key ในการตรวจสอบ
		return []byte(os.Getenv("ACCESS_SECRET_KEY")), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Invalid or expired token",
		})
	}

	// ถ้า token ถูกต้อง ก็อนุญาตให้ไปยัง Handler ถัดไป
	return c.Next()
}

func RoleRequired(requiredRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ดึง Authorization header
		tokenString := c.Get("Authorization")

		// ถ้าไม่พบ token ให้ส่งสถานะ Unauthorized กลับไป
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Authorization header missing.",
			})
		}

		// ตรวจสอบว่า token เป็น Bearer token หรือไม่
		tokenString = tokenString[len("Bearer "):]

		// Parse และตรวจสอบ token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// ตรวจสอบว่า token ใช้ method ที่ถูกต้อง
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token signing method.")
			}
			// ใช้ Secret Key ในการตรวจสอบ
			return []byte(os.Getenv("ACCESS_SECRET_KEY")), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid or expired token.",
			})
		}

		// ดึง Claims จาก token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid token claims.",
			})
		}

		// ดึง Role จาก Claims และตรวจสอบว่าตรงกับ requiredRole หรือไม่
		userRole := claims["Role"].(string)
		if userRole != requiredRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Access denied for this role.",
			})
		}

		// ถ้า Role ตรงกัน ก็อนุญาตให้ไปยัง Handler ถัดไป
		return c.Next()
	}
}
