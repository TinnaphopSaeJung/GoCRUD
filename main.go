package main

import (
	"fmt"
	"go-fiber-test/database"
	m "go-fiber-test/models"
	"go-fiber-test/routes"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func initDatabase() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		"root",
		"",
		"127.0.0.1",
		"3306",
		"golang_test",
	)
	var err error
	database.DBConn, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Database connected!")
	database.DBConn.AutoMigrate(&m.Product{}, &m.ProductImage{}, &m.User{}, &m.Order{}, &m.Item{}, &m.Session{})
	fmt.Println("AutoMigrate executed")
}

func main() {
	// โหลด .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	app := fiber.New()
	initDatabase()

	// ใช้ limiter middleware จาก go fiber
	app.Use(limiter.New(limiter.Config{
		Max:        10,
		Expiration: 30 * time.Second, // จำกัดให้สามารถส่งคำขอได้สูงสุด 10 ครั้งในทุก ๆ 30 วินาที
	}))

	routes.Routes(app)
	app.Static("/uploads", "./uploads")

	app.Listen(":3000")
}
