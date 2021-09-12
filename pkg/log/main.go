package log

import (
	"fmt"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/internal/models"
	"github.com/bufsnake/httpx/internal/modelsImpl"
	"github.com/logrusorgru/aurora"
	"math"
	"strings"
	"sync"
	"time"
)

type Log struct {
	AllHTTP    float64    // all http request count
	Percentage *float64   // http request percentage
	PerLock    sync.Mutex // Percentage lock
	Conf       config.Terminal
	Once       *bool
	StartTime  time.Time
	Silent     bool
	DB         *modelsImpl.Database
}

func (l *Log) Println(a ...interface{}) {
	if l.Silent {
		if len(a) == 5 {
			switch t := a[1].(type) {
			case string:
				t = strings.Trim(t, "[]\x1b\x5b\x39\x37\x6d\x30 ")
				fmt.Println(t)
			}
		}
	} else {
		temp := make([]interface{}, 0)
		temp = append(temp, "\r")
		temp = append(temp, a...)
		temp = append(temp, "                ")
		fmt.Println(temp...)
		l.percentage()
	}
}

func (l *Log) OutputHTML(data models.Datas) {
	// TODO: 入库
	err := l.DB.CreateDatas(&[]models.Datas{data})
	if err != nil {
		fmt.Println(err)
		return
	}
	l.percentage()
}

func (l *Log) Bar() {
	if l.Silent {
		return
	}
	for {
		percentage := math.Trunc((((*l.Percentage)/l.AllHTTP)*100)*1e2) * 1e-2
		fmt.Printf("\r %s: %.2f%% %s: %s %s: %.2fs", aurora.BrightWhite("Percentage").String(), percentage, aurora.BrightWhite("Output").String(), l.Conf.Output, aurora.BrightWhite("Time").String(), time.Now().Sub(l.StartTime).Seconds())
		time.Sleep(1 * time.Second / 10)
	}
}

func (l *Log) percentage() {
	if l.Silent {
		return
	}
	percentage := math.Trunc((((*l.Percentage)/l.AllHTTP)*100)*1e2) * 1e-2
	fmt.Printf("\r %s: %.2f%% %s: %s %s: %.2fs", aurora.BrightWhite("Percentage").String(), percentage, aurora.BrightWhite("Output").String(), l.Conf.Output, aurora.BrightWhite("Time").String(), time.Now().Sub(l.StartTime).Seconds())
}

func (l *Log) PercentageAdd() {
	if l.Silent {
		return
	}
	l.PerLock.Lock()
	defer l.PerLock.Unlock()
	*l.Percentage++
	l.percentage()
}
