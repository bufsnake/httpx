package log

import (
	"fmt"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/internal/models"
	"github.com/bufsnake/httpx/internal/modelsImpl"
	. "github.com/logrusorgru/aurora"
	"math"
	"sync"
	"time"
)

type Log struct {
	AllHTTP      float64    // all http request count
	Percentage   *float64   // http request percentage
	PerLock      sync.Mutex // Percentage lock
	Conf         config.Terminal
	StartTime    time.Time
	Silent       bool
	DB           *modelsImpl.Database
	DisplayError bool
}

func (l *Log) Println(StatusCode, URL, BodyLength, Title, CreateTime string) {
	if l.Silent {
		fmt.Println("\r" + URL)
	} else {
		fmt.Println(fmt.Sprintf("%-150s", "\r["+BrightGreen(StatusCode).String()+"] ["+BrightWhite(URL).String()+"] ["+BrightRed(BodyLength).String()+"] ["+BrightCyan(Title).String()+"] ["+BrightBlue(CreateTime).String()+"]"))
	}
}

func (l *Log) SaveData(data models.Datas) {
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
		fmt.Printf("\r%s: %.2f%% %s: %s %s: %.2fs", BrightWhite("Percentage").String(), percentage, BrightWhite("Output").String(), l.Conf.Output, BrightWhite("Time").String(), time.Now().Sub(l.StartTime).Seconds())
		time.Sleep(1 * time.Second / 10)
	}
}

func (l *Log) percentage() {
	if l.Silent {
		return
	}
	percentage := math.Trunc((((*l.Percentage)/l.AllHTTP)*100)*1e2) * 1e-2
	fmt.Printf("\r%s: %.2f%% %s: %s %s: %.2fs", BrightWhite("Percentage").String(), percentage, BrightWhite("Output").String(), l.Conf.Output, BrightWhite("Time").String(), time.Now().Sub(l.StartTime).Seconds())
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

func (l *Log) Error(a ...interface{}) {
	if !l.DisplayError {
		return
	}
	fmt.Println(a...)
}
