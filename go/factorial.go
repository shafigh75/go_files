package main

import (
	"fmt"
	"math/big"
)

func factorial(num int64) *big.Int {
	if num == 1 || num == 0 {
		return big.NewInt(num)
	}
	return big.NewInt(num).Mul(big.NewInt(num), factorial(num-1))
}
func main() {
	fmt.Println(factorial(3))
	fmt.Println(factorial(4))
	fmt.Println(factorial(500))
}
