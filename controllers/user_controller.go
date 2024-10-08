package controllers

import (
	"go-fiber-test/database"
	m "go-fiber-test/models"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func GetUsers(c *fiber.Ctx) error {
	db := database.DBConn
	var users []m.User

	db.Find(&users)
	return c.Status(200).JSON(fiber.Map{
		"data": &users,
	})
}

func UpdateUser(c *fiber.Ctx) error {
	db := database.DBConn
	userId := c.Params("userId")
	var user m.User

	if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("User not found.")
	}

	// update username
	usernameCheck := c.FormValue("Username")
	if usernameCheck != "" {
		user.Username = c.FormValue("Username")
	}

	// update password
	passwordCheck := c.FormValue("Password")
	if passwordCheck != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.FormValue("Password")), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error hashing password.")
		}
		newPassword := string(hashedPassword)

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(newPassword)); err == nil {
			return c.Status(fiber.StatusBadRequest).SendString("Please use a different password.")
		}

		user.Password = newPassword
	}

	// update Firstname
	firstNameCheck := c.FormValue("FirstName")
	if firstNameCheck != "" {
		user.FirstName = c.FormValue("FirstName")
	}

	// update Lastname
	lastNameCheck := c.FormValue("LastName")
	if lastNameCheck != "" {
		user.LastName = c.FormValue("LastName")
	}

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).SendString("Failed to update user.")
	}

	return c.Status(201).JSON(fiber.Map{
		"data":    &user,
		"message": "Updated user successfully.",
	})
}

func SoftDeleteUser(c *fiber.Ctx) error {
	db := database.DBConn
	userId := c.Params("userId")
	var user m.User

	// ตรวจสอบการมีอยู่ของ user
	if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
		return c.Status(404).SendString("User not found.")
	}
	username := user.Username

	// ลบ order ของ user คนนั้นและคืนจำนวนสินค้ากลับไปยังคลัง
	var order m.Order
	if err := db.Preload("Items").Where("Buyer = ?", userId).First(&order).Error; err != nil {
		return c.Status(404).SendString("Order not found.")
	}

	for _, item := range order.Items {
		var product m.Product
		if err := db.Where("Product_Name = ?", item.Product).First(&product).Error; err != nil {
			return c.Status(500).SendString("Product " + item.Product + " not found.")
		}

		product.Amount += item.Amount

		if err := db.Save(&product).Error; err != nil {
			return c.Status(500).SendString("Failed to update product amount in inventory.")
		}
	}

	if err := db.Unscoped().Where("order_id = ?", order.ID).Delete(&m.Item{}).Error; err != nil {
		return c.Status(500).SendString("Failed to delete order items.")
	}

	if err := db.Unscoped().Where("id = ?", order.ID).Delete(&order).Error; err != nil {
		return c.Status(500).SendString("Failed to delete order.")
	}

	// soft delete user
	if err := db.Delete(&user).Error; err != nil {
		return c.Status(500).SendString("Failed to user.")
	}

	return c.Status(200).JSON(fiber.Map{
		"message": username + " has been soft deleted.",
	})
}

func RestoreUser(c *fiber.Ctx) error {
	db := database.DBConn
	userId := c.Params("userId")
	var user m.User

	if err := db.Unscoped().Where("id = ?", userId).First(&user).Update("deleted_at", nil).Error; err != nil {
		return c.Status(500).SendString("Failed to restore user.")
	}

	return c.Status(200).JSON(fiber.Map{
		"data":    &user,
		"message": "Restore " + user.Username + " successfully.",
	})
}

func HardDeleteUser(c *fiber.Ctx) error {
	db := database.DBConn
	userId := c.Params("userId")
	var user m.User

	if err := db.Unscoped().Where("id = ? AND deleted_at IS NOT NULL", userId).First(&user).Error; err != nil {
		return c.Status(404).SendString("User not found.")
	}
	username := user.Username

	if err := db.Unscoped().Delete(&user).Error; err != nil {
		return c.Status(500).SendString("Failed to remove user.")
	}

	return c.Status(200).JSON(fiber.Map{
		"message": username + " has been deleted.",
	})
}
