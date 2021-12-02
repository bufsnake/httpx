package utils

import (
	"github.com/bufsnake/parseip"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func ICPInfo(data string) string {
	icp_info := regexp.MustCompile("\\W[ ]{0,10}[I公]{1}[ ]{0,10}[C网]{1}[ ]{0,10}[P安]{1}[ ]{0,10}[证备][ ]{0,10}\\d+[ ]{0,10}[号\\-\\d ]{0,10}")
	icp_regexp := icp_info.FindAllString(data, -1)
	icp_data := ""
	icp_ := make(map[string]bool)
	for i := 0; i < len(icp_regexp); i++ {
		icp_regexp[i] = strings.ReplaceAll(icp_regexp[i], " ", "")
		if _, ok := icp_[icp_regexp[i]]; !ok {
			icp_[icp_regexp[i]] = true
			icp_data += icp_regexp[i] + "|"
		}
	}
	return strings.Trim(icp_data, "|")
}

func IsDomain(data string) bool {
	if matched, _ := regexp.MatchString("\\d{0,3}\\.\\d{0,3}\\.\\d{0,3}\\.\\d{0,3}", data); matched {
		// 获取host部分
		host := strings.ReplaceAll(strings.ReplaceAll(data, "http://", ""), "https://", "")
		if strings.Contains(host, "/") {
			host = strings.Split(host, "/")[0]
		}
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}
		_, _, err := parseip.ParseIP(host)
		if err != nil {
			return true
		}
		return false
	}
	return true
}

func IsArrExist(datas []string, data string) bool {
	for _, data_ := range datas {
		if data_ == data {
			return true
		}
	}
	return false
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// 写文件
func AppendFile(filename, data string) error {
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	defer fd.Close()
	buf := []byte(data)
	_, err = fd.Write(buf)
	return err
}
