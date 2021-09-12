package api

import (
	"github.com/bufsnake/httpx/internal/models"
	"github.com/bufsnake/httpx/internal/modelsImpl"
	"github.com/gin-gonic/gin"
	"log"
	"strconv"
)

type api struct {
	db *modelsImpl.Database
}

func NewAPI(db *modelsImpl.Database) api {
	return api{db: db}
}

type getdata struct {
	Total int            `json:"total"`
	Datas []models.Datas `json:"datas"`
}

func (a *api) GetData(c *gin.Context) {
	page := c.Query("page")
	flag := c.Query("flag")
	page_, err := strconv.Atoi(page)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	flag_, err := strconv.Atoi(flag)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	datas, count, err := a.db.ReadDatas(page_, flag_)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.JSON(200, getdata{Datas: datas, Total: int(count)})
}

func (a *api) ImageLoad(c *gin.Context) {
	id := c.Query("id")
	atoi, err := strconv.Atoi(id)
	if err != nil {
		log.Println(err)
		return
	}
	image, err := a.db.ReadImage(atoi)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.String(200, image)
}

func (a *api) Search(c *gin.Context) {
	word := c.Query("word")
	page := c.Query("page")
	flag := c.Query("flag")
	if word == "" {
		c.String(500, "keyword is empty")
		return
	}
	page_, err := strconv.Atoi(page)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	flag_, err := strconv.Atoi(flag)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	datas, count, err := a.db.SearchDatas(word, page_, flag_)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	c.JSON(200, getdata{Datas: datas, Total: int(count)})
}

func (a *api) Copy(c *gin.Context) {
	word := c.Query("word")
	links, err := a.db.CopyLinks(word)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.String(200, links)
}
