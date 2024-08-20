package session

import (
	"go-fiber-test/database"
	m "go-fiber-test/models"
	"time"
)

func UpdateSessionActivity(userID uint) error {
	db := database.DBConn
	session := m.Session{}

	// ตรวจสอบว่ามี Session อยู่แล้วหรือไม่ หรือสร้างใหม่ถ้าไม่มี
	if err := db.Where("user_id = ?", userID).FirstOrCreate(&session, m.Session{UserID: userID}).Error; err != nil {
		return err
	}

	// update เวลาการกระทำล่าสุด
	session.LastActive = time.Now()
	return db.Save(&session).Error
}
