package modelsImpl

import (
	"github.com/bufsnake/httpx/internal/models"
)

func (d *Database) CreateLock() error {
	lock := models.Lock{Lock: false}
	return d.db.Create(&lock).Error
}

func (d *Database) ReadLock() (models.Lock, error) {
	lock := models.Lock{}
	err := d.db.Where("`id` = 1").First(&lock).Error
	return lock, err
}

func (d *Database) UpdateLock() error {
	return d.db.Model(&models.Lock{}).Where("`id` = 1").Update("lock", true).Error
}
