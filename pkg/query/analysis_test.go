package query

import (
	"fmt"
	"testing"
)

func TestAnalysisQuery(t *testing.T) {
	input := `body="xxxx" and ((ip!="x" and (ip="x" or ip="b") or title="c")) and tls~!="x"`
	input = `protocol!="<>"`
	//input := `"admin" && ip="127.0.0.1" && "test" || ip="192.168.20.1"`
	sql, params, formatInput, err := AnalysisQuery(input)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(sql)
	fmt.Println(params)
	fmt.Println(formatInput)
}

func TestNewStack(t *testing.T) {
	s := NewStack()
	s.PUSH("{")
	fmt.Println(s.POP())
	fmt.Println(s.POP())
}
