package models

import (
	"time"

	"gorm.io/gorm"
)

type ProductImage struct {
	gorm.Model
	ProductID uint   `json:"product_id"`
	ImageURL  string `json:"image_url"`
}

type Product struct {
	gorm.Model
	Product_Name string         `json:"Product_Name"`
	Price        int            `json:"Price"`
	Amount       int            `json:"Amount"`
	Images       []ProductImage `gorm:"foreignKey:ProductID" json:"Images"`
}

type Item struct {
	gorm.Model
	Product string `json:"Product"`
	Amount  int    `json:"Amount"`
	OrderID uint   // Foreign Key
}

type Order struct {
	gorm.Model
	Buyer       string `json:"Buyer"`
	Items       []Item `gorm:"foreignKey:OrderID"`
	Total_Price int    `json:"Total Price"`
}

type User struct {
	gorm.Model
	Username  string `json:"Username" validate:"required"`
	Password  string `json:"Password" validate:"required, min=6, max=20"`
	FirstName string `json:"FirstName" validate:"required"`
	LastName  string `json:"LastName" validate:"required"`
	Role      string `json:"Role"`
	Approve   bool   `json:"Approve"`
}

// ใช้ GORM BeforeCreate สำหรับตั้งค่า default ให้กับ Role และ Approve
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.Role == "" {
		u.Role = "user"
	}
	if !u.Approve {
		u.Approve = false
	}
	return
}

type Session struct {
	UserID     uint      `gorm:"primaryKey"`
	LastActive time.Time `json:"LastActive"`
}
