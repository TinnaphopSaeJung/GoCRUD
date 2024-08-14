package database

import (
	_ "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	// สร้างตัวแปรชื่อ DBConn เป็นประเภท gorm.DB เพื่อเอาไปรับการ Connect
	DBConn *gorm.DB
)
