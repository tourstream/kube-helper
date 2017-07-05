package util

import (
	"fmt"
)

func Dump(a interface{}) {
	fmt.Printf("%+v\n", a)
}
