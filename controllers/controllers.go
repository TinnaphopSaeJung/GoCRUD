package controllers

import (
	"go-fiber-test/database"
	m "go-fiber-test/models"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func GetProducts(c *fiber.Ctx) error {
	db := database.DBConn
	var products []m.Product

	db.Preload("Images").Find(&products)
	return c.Status(200).JSON(fiber.Map{
		"data":    &products,
		"message": "Show all products.",
	})
}

func GetOrders(c *fiber.Ctx) error {
	db := database.DBConn
	var orders []m.Order

	// ใช้ GORM Preloading เพื่อให้ GORM สามารถโหลดข้อมูลที่สัมพันธ์กับ Orders มาได้
	db.Preload("Items").Find(&orders)
	return c.Status(200).JSON(fiber.Map{
		"data":    &orders,
		"message": "Show all orders.",
	})
}

func GetProduct(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")
	var product m.Product

	db.Preload("Images").Where("id = ?", id).Find(&product)
	return c.Status(200).JSON(fiber.Map{
		"data":    &product,
		"message": "Show " + product.Product_Name + " success.",
	})
}

func GetOrder(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")
	var orders m.Order

	db.Preload("Items").Where("id = ?", id).Find(&orders)
	return c.Status(200).JSON(fiber.Map{
		"data":    &orders,
		"message": "Show orders successfully.",
	})
}

func AddProduct(c *fiber.Ctx) error {
	db := database.DBConn
	var product m.Product

	// อ่านค่า field ต่าง ๆ ใน Form
	product.Product_Name = c.FormValue("Product_Name")

	// อ่านค่า Price
	price, err := strconv.Atoi(c.FormValue("Price"))
	if err != nil {
		return c.Status(400).SendString("Price Invalid")
	}
	product.Price = price

	// อ่านค่า Amount
	amount, err := strconv.Atoi(c.FormValue("Amount"))
	if err != nil {
		return c.Status(400).SendString("Amount Invalid")
	}
	product.Amount = amount

	// สร้าง product ในฐานข้อมูลก่อน เพื่อให้ได้ Product ID
	if err := db.Create(&product).Error; err != nil {
		return c.Status(500).SendString("Failed to create product.")
	}

	// จัดการการ upload รูปภาพ
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).SendString("Failed to parse form data.")
	}

	files := form.File["Images"]

	// บันทึกเส้นทางไฟล์ในฐานข้อมูลในตาราง ProductImage
	for _, file := range files {
		filename := uuid.New().String() + filepath.Ext(file.Filename)
		if err := c.SaveFile(file, filepath.Join("uploads", filename)); err != nil {
			return c.Status(500).SendString("Upload Image Invalid")
		}

		// สร้าง ProductImage และเชื่อมโยงกับ Product ID
		productImage := m.ProductImage{
			ProductID: product.ID,
			ImageURL:  "/uploads/" + filename,
		}

		// บันทึก ProductImage ลงในฐานข้อมูล
		if err := db.Create(&productImage).Error; err != nil {
			return c.Status(500).SendString("Failed to save product image.")
		}
	}

	// โหลด product พร้อมกับ images
	if err := db.Preload("Images").First(&product, product.ID).Error; err != nil {
		return c.Status(500).SendString("Failed to load product with images.")
	}

	return c.Status(201).JSON(fiber.Map{
		"data":    product,
		"message": "Successfully created product.",
	})
}

func AddOrder(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")

	var orderRequest struct {
		Items []m.Item `json:"Items"`
	}

	if err := c.BodyParser(&orderRequest); err != nil {
		return c.Status(503).SendString(err.Error())
	}

	// ตรวจสอบว่า User นี้มีอยู่ระบบหรือไม่
	var user m.User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid Buyer.")
	}

	var total_price int
	var updatedItems []m.Item

	for _, item := range orderRequest.Items {
		var product m.Product
		if err := db.Where("Product_Name = ?", item.Product).First(&product).Error; err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Didn't find the product you were looking for.")
		}

		if product.Amount < item.Amount {
			return c.Status(fiber.StatusUnauthorized).SendString(
				"Product " + item.Product + " is not available in sufficient quantity.",
			)
		}

		product.Amount -= item.Amount
		if err := db.Save(&product).Error; err != nil {
			return c.Status(500).SendString("Failed to update product quantity.")
		}

		total_price += product.Price * item.Amount
		updatedItems = append(updatedItems, m.Item{
			Product: item.Product,
			Amount:  item.Amount,
		})
	}

	order := m.Order{
		Buyer:       id,
		Items:       updatedItems,
		Total_Price: total_price,
	}

	if err := db.Create(&order).Error; err != nil {
		return c.Status(500).SendString("Failed to create order.")
	}

	return c.Status(201).JSON(fiber.Map{
		"data":    order,
		"message": "Add the order you want successfully.",
	})
}

func UpdateOrder(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")

	var orderRequest struct {
		Items []m.Item `json:"Items"`
	}

	if err := c.BodyParser(&orderRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	// ตรวจสอบว่า Order ที่ต้องการ update นี้มีอยู่ในระบบหรือไม่
	var order m.Order
	if err := db.Preload("Items").Where("id = ?", id).First(&order).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Order not found.")
	}

	var total_price int
	var total_price_nonupdate int
	var updatedItems []m.Item

	// สร้าง mapping สำหรับเก็บรายการ order ก่อนที่จะถูก update
	originalItems := make(map[string]m.Item)
	for _, item := range order.Items {
		originalItems[item.Product] = item
	}

	// สร้าง set โดยให้ product ที่อยู่ใน request เป็น true ทั้งหมด
	updatedProducts := make(map[string]bool)
	for _, item := range orderRequest.Items {
		updatedProducts[item.Product] = true
	}

	for productName, item := range originalItems {
		// ตรวจสอบสินค้าที่ไม่ได้อยู่ใน request ==> updatedProducts[productName] == false
		if !updatedProducts[productName] {
			var product m.Product
			if err := db.Where("Product_Name = ?", productName).First(&product).Error; err != nil {
				return c.Status(500).SendString("Failed to retrieve product " + productName + " from the database.")
			}
			log.Printf("Item not included in update: Product: %s, Amount: %d, Price: %d", item.Product, item.Amount, product.Price)
			total_price_nonupdate += product.Price * item.Amount

			updatedItems = append(updatedItems, item)
		}
	}

	// อัปเดตข้อมูล product จาก order ใหม่ที่ update มา
	for _, item := range orderRequest.Items {
		var product m.Product
		if err := db.Where("Product_Name = ?", item.Product).First(&product).Error; err != nil {
			return c.Status(404).SendString("Product " + item.Product + " not found.")
		}

		// ตรวจสอบจำนวนสินค้า
		originalAmount := originalItems[item.Product].Amount
		if product.Amount+originalAmount < item.Amount {
			return c.Status(400).SendString(
				"Product " + item.Product + " is not available in sufficient quantity.",
			)
		}

		// Update จำนวนสินค้าใน Product
		product.Amount = product.Amount + originalAmount - item.Amount
		if err := db.Save(&product).Error; err != nil {
			return c.Status(500).SendString("Failed to update product quantity.")
		}

		// คำนวณราคาใหม่
		total_price += product.Price * item.Amount

		if originalItems, exists := originalItems[item.Product]; exists {
			// ถ้า item นี้มีอยู่แล้วใน order, ให้ update จำนวนสินค้า
			originalItems.Amount = item.Amount
			if err := db.Save(&originalItems).Error; err != nil {
				return c.Status(500).SendString("Failed to update item.")
			}
			updatedItems = append(updatedItems, originalItems)
		} else {
			// ถ้า item นี้ยังไม่มีอยู่ใน order, ให้สร้างใหม่
			newItem := m.Item{
				Product: item.Product,
				Amount:  item.Amount,
				OrderID: order.ID,
			}
			if err := db.Create(&newItem).Error; err != nil {
				return c.Status(500).SendString("Failed to add new item.")
			}
			updatedItems = append(updatedItems, newItem)
		}
	}

	// อัปเดตรายการสินค้าใน order
	order.Items = updatedItems
	order.Total_Price = total_price + total_price_nonupdate

	if err := db.Save(&order).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to update order.")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":    order,
		"message": "Order has been successfully updated.",
	})
}

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

	// สร้าง Payload สำหรับ JWT Token
	claim := jwt.MapClaims{
		"Username": user.Username,
		"Role":     user.Role,
		"UserID":   user.ID,
	}

	// ดึง secretKey จาก environment variable
	secretKey := os.Getenv("SECRET_KEY")

	// สร้าง JWT Token
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claim).SignedString([]byte(secretKey))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error creating token.")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Hello " + user.Username + ", You logged in Successfully.",
		"payload": claim,
		"token":   token,
		"userId":  user.ID,
	})
}

func UpdateProduct(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")
	var product m.Product

	// ค้นหา product เดิมในฐานข้อมูล
	if err := db.Preload("Images").Where("id = ?", id).First(&product).Error; err != nil {
		return c.Status(404).SendString("Product not Found")
	}

	// Update field ต่าง ๆ
	// Update Product_Name
	nameCheck := c.FormValue("Product_Name")
	if nameCheck != "" {
		product.Product_Name = c.FormValue("Product_Name")
	}

	// Update Price
	priceCheck := c.FormValue("Price")
	if priceCheck != "" {
		price, err := strconv.Atoi(c.FormValue("Price"))
		if err != nil {
			return c.Status(400).SendString("Price Invalid")
		}
		product.Price = price
	}

	// Update Amount
	amountCheck := c.FormValue("Amount")
	if amountCheck != "" {
		amount, err := strconv.Atoi(c.FormValue("Amount"))
		if err != nil {
			return c.Status(400).SendString("Amount Invalid")
		}
		product.Amount = amount
	}

	// จัดการการ upload รูปภาพใหม่ (ถ้ามี)
	form, err := c.MultipartForm()
	// ตรวจสอบว่ามีไฟล์ใหม่ถูก upload มาไหม
	if err == nil && form != nil {
		// ตรวจสอบว่ามีไฟล์ใหม่ถูก upload มาไหม
		if len(form.File["Images"]) > 0 {
			// ลบรูปภาพเก่าทั้งหมดก่อน (ถ้ามี)
			for _, img := range product.Images {
				oldImagePath := "." + img.ImageURL
				if err := os.Remove(oldImagePath); err != nil {
					return c.Status(500).SendString("Failed to remove old image.")
				}
			}

			// ลบข้อมูลรูปภาพจากฐานข้อมูล
			if err := db.Where("product_id = ?", product.ID).Delete(&m.ProductImage{}).Error; err != nil {
				return c.Status(500).SendString("Failed to remove image data.")
			}

			// บันทึกรูปภาพใหม่
			files := form.File["Images"]
			for _, file := range files {
				filename := uuid.New().String() + filepath.Ext(file.Filename)
				if err := c.SaveFile(file, filepath.Join("uploads", filename)); err != nil {
					return c.Status(500).SendString("Failed to upload new image.")
				}

				// บันทึกเส้นทางไฟล์ใหม่ในฐานข้อมูล
				productImage := m.ProductImage{
					ProductID: product.ID,
					ImageURL:  "/uploads/" + filename,
				}

				// บันทึก ProductImage ลงในฐานข้อมูล
				if err := db.Create(&productImage).Error; err != nil {
					return c.Status(500).SendString("Failed to save product image.")
				}
			}
		}
	}

	// บันทึกการเปลี่ยนแปลงในฐานข้อมูล
	if err := db.Save(&product).Error; err != nil {
		return c.Status(500).SendString("Failed to update product.")
	}

	// โหลด product พร้อมกับ images ที่อัปเดตแล้ว
	if err := db.Preload("Images").First(&product, product.ID).Error; err != nil {
		return c.Status(500).SendString("Failed to load updated product with images.")
	}

	return c.Status(201).JSON(fiber.Map{
		"data":    product,
		"message": product.Product_Name + " has been successfully updated.",
	})
}

func RemoveProduct(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")
	var product m.Product

	// ค้นหา product ที่ต้องการลบ
	if err := db.Preload("Images").Where("id = ?", id).First(&product).Error; err != nil {
		return c.Status(404).SendString("Product not found.")
	}

	// ลบรูปภาพออกจากระบบ
	for _, img := range product.Images {
		imagePath := "." + img.ImageURL
		if err := os.Remove(imagePath); err != nil {
			return c.Status(500).SendString("Failed to remove image.")
		}
	}

	// ลบข้อมูลรูปภาพจากฐานข้อมูล
	if err := db.Where("product_id = ?", product.ID).Delete(&m.ProductImage{}).Error; err != nil {
		return c.Status(500).SendString("Failed to remove image data.")
	}

	productName := product.Product_Name

	result := db.Delete(&product, id)
	if result.RowsAffected == 0 {
		return c.SendStatus(404)
	}

	return c.Status(201).JSON(fiber.Map{
		"message": productName + " has been successfully deleted.",
	})
}

func RemoveOrder(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("id")
	var order m.Order

	// ตรวจสอบว่า Order ที่ต้องการลบมีอยู่ในฐานข้อมูลหรือไม่
	if err := db.Preload("Items").Where("id = ?", id).First(&order).Error; err != nil {
		return c.Status(404).SendString("Order not found.")
	}

	// ลบรายการสินค้าในคำสั่งซื้อนั้น
	if err := db.Where("order_id = ?", id).Delete(&m.Item{}).Error; err != nil {
		return c.Status(500).SendString("Failed to delete order items.")
	}

	// ลบคำสั่งซื้อ
	result := db.Delete(&order, id)
	if result.RowsAffected == 0 {
		return c.SendStatus(404)
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "Order has been successfully deleted.",
	})
}
