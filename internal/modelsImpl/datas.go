package modelsImpl

import (
	"encoding/base64"
	"fmt"
	"github.com/bufsnake/httpx/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Database struct {
	db     *gorm.DB
	server bool
}

func NewDatabase(dbname string, runserver bool) (Database, error) {
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
	return Database{db: db, server: runserver}, err
}

func (d *Database) InitDatabase() error {
	err := d.db.AutoMigrate(&models.Images{})
	if err != nil {
		return err
	}
	err = d.db.AutoMigrate(&models.Lock{})
	if err != nil {
		return err
	}
	err = d.db.AutoMigrate(&models.Finger{})
	if err != nil {
		return err
	}
	err = d.CreateLock()
	if err != nil {
		return err
	}
	lock, err := d.ReadLock()
	if err != nil {
		return err
	}
	if !lock.Lock && d.server {
		// 更新数据库顺序
		datas := make([]models.Datas, 0)
		datas, err = d.ReadAllDatas()
		if err != nil {
			return err
		}
		sort_arra := make([]string, 0)
		data_temp := make(map[string]models.Datas)
		m := make(map[string]bool)
		for i := 0; i < len(datas); i++ {
			parse, _ := url.Parse(datas[i].URL)
			unique_id := parse.Host + "--" + parse.Path + "--" + parse.RawQuery + "--" + parse.Scheme
			if _, ok := m[unique_id]; !ok {
				m[unique_id] = true
				sort_arra = append(sort_arra, unique_id)
				datas[i].Id = 0
				data_temp[unique_id] = datas[i]
			}
		}
		datas = make([]models.Datas, 0)
		sort.Strings(sort_arra)
		for i := 0; i < len(sort_arra); i++ {
			datas = append(datas, data_temp[sort_arra[i]])
		}
		err = d.DeleteDatas()
		if err != nil {
			return err
		}
		count := len(datas) / 500
		count_ := len(datas) % 500
		var i = 0
		for i = 0; i < count; i++ {
			err = d.ReCreateDatas(datas[i*500 : (i+1)*500])
			if err != nil {
				return err
			}
		}
		if count_ > 0 {
			err = d.ReCreateDatas(datas[i*500:])
			if err != nil {
				return err
			}
		}
		// 更新完修改为true
		err = d.UpdateLock()
		if err != nil {
			return err
		}
		log.Println("re gen database success")
	}
	return d.db.AutoMigrate(&models.Datas{})
}

func (d *Database) DeleteDatas() error {
	return d.db.Where("id <> ?", -1).Delete(&models.Datas{}).Error
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

func (d *Database) CreateFinger(datas *[]models.Finger) error {
	return d.db.Create(datas).Error
}

func (d *Database) ReCreateDatas(datas []models.Datas) error {
	return d.db.Create(&datas).Error
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

func (d *Database) ReadAllDatas() (datas []models.Datas, err error) {
	err = d.db.Model(&models.Datas{}).Find(&datas).Error
	return
}

func (d *Database) UpdateDatas(datas *[]models.Datas) error {
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

func (d *Database) SearchDatas(sql string, params []interface{}, page, flag int) ([]models.Datas, int64, error) {
	datas := make([]models.Datas, 0)
	err := d.db.Model(&models.Datas{}).Limit(flag).Offset((page-1)*flag).Where(sql, params...).Find(&datas).Error
	var count int64
	d.db.Debug().Model(&models.Datas{}).Where(sql, params...).Count(&count)
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

func (d *Database) CopyLinks(sql string, params []interface{}) (string, error) {
	datas := make([]models.Datas, 0)
	var err error
	if sql != "" {
		err = d.db.Model(&models.Datas{}).Where(sql, params...).Find(&datas).Error
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

func (d *Database) ReadFinger(url string) ([]models.Finger, error) {
	fingers := make([]models.Finger, 0)
	err := d.db.Model(&models.Finger{}).Where("url = ?", url).Find(&fingers).Error
	if err != nil {
		return fingers, err
	}
	return fingers, nil
}
