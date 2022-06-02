package modelsImpl

import (
	"github.com/bufsnake/httpx/internal/models"
)

func (d *Database) CreateLock() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	lock := models.Lock{Lock: false}
	return d.db.Create(&lock).Error
}

func (d *Database) ReadLock() (models.Lock, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	lock := models.Lock{}
	err := d.db.Where("`id` = 1").First(&lock).Error
	return lock, err
}

func (d *Database) UpdateLock() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.db.Model(&models.Lock{}).Where("`id` = 1").Update("lock", true).Error
}
