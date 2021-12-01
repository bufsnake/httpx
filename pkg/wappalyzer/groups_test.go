package wappalyzer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
)

func TestNewGroups(t *testing.T) {
	groups := "wappalyzer/src/groups.json"
	groups_ := make(Groups)
	count := 0
	file_content, err := os.ReadFile(groups)
	if err != nil {
		log.Println(err)
		return
	}
	err = json.Unmarshal(file_content, &groups_)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	for k, v := range groups_ {
		count++
		fmt.Println(k, v["name"])
	}
	fmt.Println(count)
}
