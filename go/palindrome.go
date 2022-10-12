package main

import (
	"strconv"
)

func isPalindrome(x int) bool {
	t := strconv.Itoa(x)
	count := len(t)
	arr := make([]int, 1000)
	for index, num := range t {
		number := int(num - '0')
		arr[index] = number
	}
	for i, j := 0, count-1; i < count/2; i, j = i+1, j-1 {
		if arr[i] != arr[j] {
			return false
		}
	}
	return true
}

func main() {
	print(isPalindrome(1378731))
}
