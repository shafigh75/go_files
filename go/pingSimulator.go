package main

import (
	"fmt"
	"strconv"
	"time"
)

func main() {
	a := make(chan string, 1)
	try := 0
	for {
		go func() {
			a <- "pinging server: try number [" + strconv.Itoa(try) + "]"
			try++
		}()
		time.Sleep(time.Second)
		fmt.Println(<-a)
	}
}
