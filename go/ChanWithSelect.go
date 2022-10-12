package main

import (
	"fmt"
	"sync"
	"time"
)

func recv(ch chan string) {
	for {
		select {
		case data := <-ch:
			fmt.Printf("text recieve from channel: %s \n", data)
			return
		default:
			print("waiting for a data over channel ... \n")
			time.Sleep(1 * time.Second)
		}
	}

}

func send(ch chan string, text string) {
	time.Sleep(5 * time.Second)
	ch <- text
	fmt.Printf("text sent to channel : %s \n", text)
	close(ch)

}

func main() {
	var wg sync.WaitGroup
	channel := make(chan string)
	wg.Add(2)
	go func() {
		defer wg.Done()
		send(channel, "My name is mohammad!")
	}()
	go func() {
		defer wg.Done()
		recv(channel)
	}()
	wg.Wait()
	print("DONE")

}

