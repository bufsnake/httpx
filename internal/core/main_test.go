package core

import (
	"fmt"
	"testing"
)

func TestNewCore(t *testing.T) {
	path := parsePath("https://www.baidu.com", "//fsadfas/fasfasffasdf/fasdfas/admin/admin.text/admin.php")
	fmt.Println(path)
}
