package controllers

import (
	"go-fiber-test/database"
	m "go-fiber-test/models"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

func GetProduct(c *fiber.Ctx) error {
	db := database.DBConn
	productId := c.Params("productId")
	var product m.Product

	db.Preload("Images").Where("id = ?", productId).Find(&product)
	return c.Status(200).JSON(fiber.Map{
		"data":    &product,
		"message": "Show " + product.Product_Name + " success.",
	})
}

func GetProductImage(c *fiber.Ctx) error {
	db := database.DBConn
	productID := c.Params("product_id")
	imageID := c.Params("image_id")

	var product m.Product
	if err := db.Preload("Images").Where("id = ?", productID).First(&product).Error; err != nil {
		return c.Status(404).SendString("Product not found.")
	}

	var images m.ProductImage
	if err := db.Where("id = ? AND product_id =?", imageID, productID).First(&images).Error; err != nil {
		return c.Status(404).SendString("Image not found.")
	}

	return c.Status(200).JSON(fiber.Map{
		"data":    &images,
		"message": "Show images of " + product.Product_Name,
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
		"data":    &product,
		"message": "Successfully created product.",
	})
}

func UpdateProduct(c *fiber.Ctx) error {
	db := database.DBConn
	productId := c.Params("productId")
	var product m.Product

	// ค้นหา product เดิมในฐานข้อมูล
	if err := db.Preload("Images").Where("id = ?", productId).First(&product).Error; err != nil {
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

	// Upload Images
	form, err := c.MultipartForm()
	// ตรวจสอบว่ามีไฟล์ใหม่ถูก upload มาไหม
	if err == nil && form != nil {
		// ตรวจสอบว่ามีไฟล์ใหม่ถูก upload มาไหม
		if len(form.File["Images"]) > 0 {
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

				product.Images = append(product.Images, productImage)
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
		"data":    &product,
		"message": product.Product_Name + " has been successfully updated.",
	})
}

func SoftDeleteProduct(c *fiber.Ctx) error {
	db := database.DBConn
	productId := c.Params("productId")
	var product m.Product

	if err := db.Preload("Images").Where("id = ?", productId).First(&product).Error; err != nil {
		return c.Status(404).SendString("Product not found.")
	}

	// soft delete images ในฐานข้อมูล
	if err := db.Where("product_id = ?", product.ID).Delete(&m.ProductImage{}).Error; err != nil {
		return c.Status(500).SendString("Failed to remove image data.")
	}

	productName := product.Product_Name

	// soft delete product
	if err := db.Where("id = ?", productId).Delete(&product).Error; err != nil {
		return c.Status(500).SendString("Failed to delete product.")
	}

	return c.Status(201).JSON(fiber.Map{
		"message": productName + " has been successfully deleted.",
	})
}

func RestoreProduct(c *fiber.Ctx) error {
	db := database.DBConn
	productId := c.Params("productId")
	var product m.Product

	if err := db.Unscoped().Preload("Images").Where("id = ?", productId).First(&product).Error; err != nil {
		return c.Status(404).SendString("Can't find the product you want to restore.")
	}

	// restore images ในฐานข้อมูล
	for _, image := range product.Images {
		if err := db.Unscoped().Where("product_id = ?", product.ID).First(&m.ProductImage{}).Update("deleted_at", nil).Error; err != nil {
			return c.Status(500).SendString("Failed to restore images : " + strconv.Itoa(int(image.ID)))
		}
	}

	// restore product
	if err := db.Unscoped().Where("id = ?", productId).First(&product).Update("deleted_at", nil).Error; err != nil {
		return c.Status(500).SendString("Failed to restore product.")
	}

	return c.Status(201).JSON(fiber.Map{
		"data":    &product,
		"message": "Restore " + product.Product_Name + " successfully.",
	})
}

func HardDeleteProduct(c *fiber.Ctx) error {
	db := database.DBConn
	productId := c.Params("productId")
	var product m.Product

	if err := db.Unscoped().Preload("Images").Where("id = ? AND deleted_at IS NOT NULL", productId).First(&product).Error; err != nil {
		return c.Status(404).SendString("Product not found.")
	}

	// ลบรูปภาพออกจากระบบ (ลบใน folder uploads)
	for _, img := range product.Images {
		imagePath := "." + img.ImageURL
		if err := os.Remove(imagePath); err != nil {
			return c.Status(500).SendString("Failed to remove image.")
		}
	}

	// hard delete images ในฐานข้อมูล
	if err := db.Unscoped().Where("product_id = ? AND deleted_at IS NOT NULL", product.ID).Delete(&m.ProductImage{}).Error; err != nil {
		return c.Status(500).SendString("Failed to remove image data.")
	}

	productName := product.Product_Name

	if err := db.Unscoped().Delete(&product).Error; err != nil {
		return c.Status(500).SendString("Failed to remove product.")
	}

	return c.Status(201).JSON(fiber.Map{
		"message": productName + " has been successfully deleted.",
	})
}

func RemoveImage(c *fiber.Ctx) error {
	db := database.DBConn
	productID := c.Params("product_id")
	imageID := c.Params("image_id")

	var product m.Product
	if err := db.Preload("Images").Where("id = ?", productID).First(&product).Error; err != nil {
		return c.Status(404).SendString("Product not found.")
	}

	var image m.ProductImage
	if err := db.Where("id = ? AND product_id = ?", imageID, productID).First(&image).Error; err != nil {
		return c.Status(404).SendString("Image not found.")
	}

	// ลบรูปภาพออกจากระบบ (ลบใน folder uploads)
	imagePath := "." + image.ImageURL
	if err := os.Remove(imagePath); err != nil {
		return c.Status(500).SendString("Failed to remove image file.")
	}

	// hard delete images ในฐานข้อมูล
	if err := db.Unscoped().Delete(&image).Error; err != nil {
		return c.Status(500).SendString("Failed to delete image record.")
	}

	if err := db.Preload("Images").First(&product, product.ID).Error; err != nil {
		return c.Status(500).SendString("Failed to load updated product with images.")
	}

	return c.Status(200).JSON(fiber.Map{
		"data":    &product,
		"message": "Image has been successfully removed.",
	})
}
