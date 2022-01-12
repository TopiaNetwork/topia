package common

import "fmt"

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args))
}
