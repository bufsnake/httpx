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
	AllHTTP        float64 // all http request count
	NumberOfAssets int
	Percentage     *float64   // http request percentage
	PerLock        sync.Mutex // Percentage lock
	Conf           *config.Terminal
	StartTime      time.Time
	Silent         bool
	DB             *modelsImpl.Database
	DisplayError   bool
	numberOfTabs   int
}

func (l *Log) SetNumberOfTabs(count int) {
	l.numberOfTabs = count
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
	for {
		l.percentage()
		time.Sleep(1 * time.Second / 10)
	}
}

func (l *Log) percentage() {
	if l.Silent {
		return
	}
	percentage := math.Trunc((((*l.Percentage)/l.AllHTTP)*100)*1e2) * 1e-2
	if *l.Conf.Stop {
		fmt.Printf("\r%s: %d %s: %d %s: %.2f%% %s: %s %s: %.2fs Wait To Stop...", BrightWhite("NumberOfTabs").String(), l.numberOfTabs, BrightWhite("NumberOfAssets").String(), l.NumberOfAssets, BrightWhite("Percentage").String(), percentage, BrightWhite("Output").String(), l.Conf.Output, BrightWhite("Time").String(), time.Now().Sub(l.StartTime).Seconds())
		return
	}
	fmt.Printf("\r%s: %d %s: %d %s: %.2f%% %s: %s %s: %.2fs", BrightWhite("NumberOfTabs").String(), l.numberOfTabs, BrightWhite("NumberOfAssets").String(), l.NumberOfAssets, BrightWhite("Percentage").String(), percentage, BrightWhite("Output").String(), l.Conf.Output, BrightWhite("Time").String(), time.Now().Sub(l.StartTime).Seconds())
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
