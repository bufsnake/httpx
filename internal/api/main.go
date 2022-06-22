package api

import (
	"github.com/bufsnake/httpx/internal/models"
	"github.com/bufsnake/httpx/internal/modelsImpl"
	"github.com/bufsnake/query"
	"github.com/bufsnake/wappalyzer"
	"github.com/gin-gonic/gin"
	"log"
	"strconv"
	"strings"
)

type api struct {
	db *modelsImpl.Database
}

func NewAPI(db *modelsImpl.Database) *api {
	err := query.AddKeywords([]string{
		"ip", "host", "title", "statuscode", "bodylength", "createtime",
		"body", "tls", "icp",
	})
	if err != nil {
		log.Fatalln(err)
	}
	return &api{db: db}
}

type getdata struct {
	Query string         `json:"query"`
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
	for i := 0; i < len(datas); i++ {
		finger, err := a.db.ReadFinger(datas[i].URL)
		if err != nil {
			continue
		}
		datas[i].Fingers = finger
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

type Query struct {
	Word string `json:"word"`
}

func (a *api) Search(c *gin.Context) {
	q := Query{}
	err := c.ShouldBindJSON(&q)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	page := c.Query("page")
	flag := c.Query("flag")
	if q.Word == "" {
		c.String(500, "keyword is empty")
		return
	}
	sql, params, formatQuery, err := query.AnalyseQuery(q.Word)
	if err != nil {
		c.String(500, "have an error in your query syntax")
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
	datas, count, err := a.db.SearchDatas(sql, params, page_, flag_)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	for i := 0; i < len(datas); i++ {
		finger, err := a.db.ReadFinger(datas[i].URL)
		if err != nil {
			continue
		}
		datas[i].Fingers = finger
	}
	c.JSON(200, getdata{Query: formatQuery, Datas: datas, Total: int(count)})
}

func (a *api) Copy(c *gin.Context) {
	q := Query{}
	var err error
	err = c.ShouldBindJSON(&q)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	links := ""
	if q.Word == "" {
		links, err = a.db.CopyLinks(q.Word, nil)
		if err != nil {
			c.String(500, err.Error())
			return
		}
	} else {
		sql, params, _, err := query.AnalyseQuery(q.Word)
		if err != nil {
			c.String(500, "have an error in your query syntax")
			return
		}
		links, err = a.db.CopyLinks(sql, params)
		if err != nil {
			c.String(500, err.Error())
			return
		}
	}
	c.String(200, links)
}

func (a *api) FingerLoad(c *gin.Context) {
	url := c.Query("url")
	finger, err := a.db.ReadFinger(url)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.JSON(200, finger)
}

func (a *api) GetICON(c *gin.Context) {
	icon := c.Query("icon")
	readICON := wappalyzer.ReadICON(icon)
	if strings.HasSuffix(icon, "svg") {
		c.Header("Content-Type", "image/svg+xml")
	}
	c.String(200, readICON)
}
