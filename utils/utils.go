package utils

import "fmt"

// StructToString is a method that prints all the struct parameters as a string
func StructToString(structVar interface{}) {
	fmt.Printf("%+v\n", structVar)
}
