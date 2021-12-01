package wappalyzer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
)

func TestNewCategories(t *testing.T) {
	categories := "wappalyzer/src/categories.json"
	categories_ := make(Categories)
	count := 0
	file_content, err := os.ReadFile(categories)
	if err != nil {
		log.Println(err)
		return
	}
	err = json.Unmarshal(file_content, &categories_)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	for k, v := range categories_ {
		count++
		fmt.Println(k, v.Name, v.Priority, v.Groups)
	}
	fmt.Println(count)
}
