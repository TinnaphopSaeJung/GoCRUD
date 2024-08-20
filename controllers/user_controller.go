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
	id := c.Params("id")
	var user m.User

	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
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
	id := c.Params("id")
	var user m.User

	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		return c.Status(404).SendString("User not found.")
	}
	username := user.Username

	if err := db.Delete(&user).Error; err != nil {
		return c.Status(500).SendString("Failed to delete image.")
	}

	return c.Status(200).JSON(fiber.Map{
		"message": username + " has been deleted.",
	})
}

func RestoreUser(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")
	var user m.User

	if err := db.Unscoped().Where("id = ?", id).First(&user).Update("deleted_at", nil).Error; err != nil {
		return c.Status(500).SendString("Failed to restore user.")
	}

	return c.Status(200).JSON(fiber.Map{
		"data":    &user,
		"message": "Restore " + user.Username + " successfully.",
	})
}

func HardDeleteUser(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")
	var user m.User

	if err := db.Unscoped().Where("id = ? AND deleted_at IS NOT NULL", id).First(&user).Error; err != nil {
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
