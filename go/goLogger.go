package main

import (
	"log"
	"time"
)

func main() {
	a := make(chan string)
	go func() {
		for {
			a <- "hello"
			time.Sleep(time.Second)
		}
	}()
	for {
		log.Println(<-a)
	}

}
