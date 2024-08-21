package controllers

import (
	"go-fiber-test/database"
	m "go-fiber-test/models"
	"log"

	"github.com/gofiber/fiber/v2"
)

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

func GetOrder(c *fiber.Ctx) error {
	db := database.DBConn
	id := c.Params("userId")
	var orders []m.Order

	db.Preload("Items").Where("Buyer = ?", id).Find(&orders)
	return c.Status(200).JSON(fiber.Map{
		"data":    &orders,
		"message": "Show orders successfully.",
	})
}

func AddOrder(c *fiber.Ctx) error {
	db := database.DBConn
	userId := c.Params("userId")

	var orderRequest struct {
		Items []m.Item `json:"Items"`
	}

	if err := c.BodyParser(&orderRequest); err != nil {
		return c.Status(503).SendString(err.Error())
	}

	// ตรวจสอบว่า User นี้มีอยู่ระบบหรือไม่
	var user m.User
	if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
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
		Buyer:       userId,
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
	orderId := c.Params("orderId")

	var orderRequest struct {
		Items []m.Item `json:"Items"`
	}

	if err := c.BodyParser(&orderRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	// ตรวจสอบว่า Order ที่ต้องการ update นี้มีอยู่ในระบบหรือไม่
	var order m.Order
	if err := db.Preload("Items").Where("id = ?", orderId).First(&order).Error; err != nil {
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
			return c.Status(500).SendString("Product " + item.Product + " not found.")
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

func RemoveOrder(c *fiber.Ctx) error {
	db := database.DBConn
	orderId := c.Params("orderId")
	var order m.Order

	// ตรวจสอบว่า Order ที่ต้องการลบมีอยู่ในฐานข้อมูลหรือไม่
	if err := db.Preload("Items").Where("id = ?", orderId).First(&order).Error; err != nil {
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

	// ลบรายการสินค้าในคำสั่งซื้อนั้น
	if err := db.Unscoped().Where("order_id = ?", orderId).Delete(&m.Item{}).Error; err != nil {
		return c.Status(500).SendString("Failed to delete order items.")
	}

	// ลบคำสั่งซื้อ
	if err := db.Unscoped().Delete(&order, orderId).Error; err != nil {
		return c.Status(500).SendString("Failed to delete order.")
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "Order has been successfully deleted.",
	})
}
