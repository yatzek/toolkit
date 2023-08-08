package main

import (
	"fmt"

	"github.com/yatzek/toolkit"
)

func main() {
	var tools toolkit.Tools

	s := tools.RandomString(10)
	fmt.Println(s)
}
