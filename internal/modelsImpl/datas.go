package modelsImpl

import (
	"encoding/base64"
	"fmt"
	"github.com/bufsnake/httpx/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Database struct {
	db *gorm.DB
}

func NewDatabase(dbname string) (Database, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             3600 * time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s", dbname)), &gorm.Config{
		CreateBatchSize: 2000,
		Logger:          newLogger,
	})
	if err != nil {
		return Database{}, err
	}
	return Database{db: db}, err
}

func (d *Database) InitDatabase() error {
	err := d.db.AutoMigrate(&models.Images{})
	if err != nil {
		return err
	}
	return d.db.AutoMigrate(&models.Datas{})
}

func (d *Database) CreateDatas(datas *[]models.Datas) error {
	if (*datas)[0].Image != "" {
		image, err := d.CreateImage(&[]models.Images{
			{Image: (*datas)[0].Image},
		})
		if err != nil {
			return err
		}
		(*datas)[0].Image = "/v1/imageload?id=" + strconv.Itoa(image)
	}
	return d.db.Create(datas).Error
}

func (d *Database) ReadDatas(page, flag int) (datas []models.Datas, count int64, err error) {
	err = d.db.Model(&models.Datas{}).Where("id between ? and ?", (page-1)*flag+1, page*flag).Find(&datas).Error
	d.db.Model(&models.Datas{}).Count(&count)
	compile := regexp.MustCompile("Subject: .*")
	for i := 0; i < len(datas); i++ {
		if datas[i].TLS == "" {
			continue
		}
		allstr := compile.FindAllString(datas[i].TLS, -1)
		body := datas[i].HTTPDump
		datas[i].HTTPDump = ""
		for j := 0; j < len(allstr); j++ {
			datas[i].HTTPDump += allstr[j] + "\n"
		}
		datas[i].HTTPDump += "\n" + body
	}
	return
}

func (d *Database) UpdateDatas(datas *[]models.Datas) error {
	return d.db.Create(datas).Error
}

func (d *Database) DeleteDatas(datas *[]models.Datas) error {
	return d.db.Create(datas).Error
}

func (d *Database) CreateImage(images *[]models.Images) (int, error) {
	create := d.db.Create(images)
	return (*images)[0].Id, create.Error
}

func (d *Database) ReadImage(id int) (string, error) {
	images := models.Images{}
	err := d.db.Model(&models.Images{}).Where("id = ?", id).Find(&images).Error
	if err != nil {
		return "", err
	}
	ima, err := base64.StdEncoding.DecodeString(images.Image)
	if err != nil {
		return "", err
	}
	return string(ima), nil
}

func (d *Database) SearchDatas(word string, page, flag int) ([]models.Datas, int64, error) {
	word = "%" + word + "%"
	datas := make([]models.Datas, 0)
	err := d.db.Model(&models.Datas{}).Limit(flag).Offset((page-1)*flag).Where("`url` LIKE ? or `title` LIKE ? or `httpdump` LIKE ? or `tls` LIKE ? or `icp` LIKE ?", word, word, word, word, word).Find(&datas).Error
	var count int64
	d.db.Model(&models.Datas{}).Where("`url` LIKE ? or `title` LIKE ? or `httpdump` LIKE ? or `tls` LIKE ? or `icp` LIKE ?", word, word, word, word, word).Count(&count)
	compile := regexp.MustCompile("Subject: .*")
	for i := 0; i < len(datas); i++ {
		if datas[i].TLS == "" {
			continue
		}
		allstr := compile.FindAllString(datas[i].TLS, -1)
		body := datas[i].HTTPDump
		datas[i].HTTPDump = ""
		for j := 0; j < len(allstr); j++ {
			datas[i].HTTPDump += allstr[j] + "\n"
		}
		datas[i].HTTPDump += "\n" + body
	}
	return datas, count, err
}

func (d *Database) CopyLinks(word string) (string, error) {
	datas := make([]models.Datas, 0)
	var err error
	if word != "" {
		word = "%" + word + "%"
		err = d.db.Model(&models.Datas{}).Where("`url` LIKE ? or `title` LIKE ? or `httpdump` LIKE ? or `tls` LIKE ? or `icp` LIKE ?", word, word, word, word, word).Find(&datas).Error
	} else {
		err = d.db.Model(&models.Datas{}).Find(&datas).Error
	}
	if err != nil {
		return "", err
	}
	data := ""
	for i := 0; i < len(datas); i++ {
		data += datas[i].URL + "\n"
	}
	return strings.Trim(data, "\n"), nil
}
