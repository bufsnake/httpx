package wappalyzer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
)

func TestNewWappalyzer(t *testing.T) {
	technologies := "wappalyzer/src/technologies/"
	schemas = make(Schema)
	count := 0
	for i := 0; i < 27; i++ {
		var chr = string(rune(96 + i))
		if chr == "`" {
			chr = "_"
		}
		file_name := technologies + chr + ".json"
		file_content, err := os.ReadFile(file_name)
		if err != nil {
			log.Println(err)
			continue
		}
		//fmt.Println(file_name, strings.Contains(string(file_content), "meta"))
		var schema Schema
		err = json.Unmarshal(file_content, &schema)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		for k, v := range schema {
			count++
			schemas[k] = v
		}
	}

	fmt.Println("total finger", count)
	for name, v := range schemas {
		_, err := TypeTest(v.Implies)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.Requires)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.RequiresCategory)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.Excludes)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.DOM)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.DNS)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.HTML)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.TEXT)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.CSS)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.Robots)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.URL)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.XHR)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.Meta)
		if err != nil {
			fmt.Println(name, err)
		}
		_, err = TypeTest(v.ScriptSrc)
		if err != nil {
			fmt.Println(name, err)
		}
	}
}
