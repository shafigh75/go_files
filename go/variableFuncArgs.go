package main

import "fmt"

func S(num ...int) int {
	sum := 0
	for _, i := range num {
		sum += i
	}
	return sum
}

func main() {
	new := []int{1, 3}
	res := S(new...)
	fmt.Println(res)
}
