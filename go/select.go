// You can edit this code!
// Click here and start typing.
package main

import (
	"fmt"
	"time"
)

func send(name string, ch chan string) {
	if name == "ch1" {
		time.Sleep(time.Second * 5)
		ch <- "hello"
		return
	}
	time.Sleep(time.Second * 2)
	ch <- "hallo"
	return
}
func main() {
	ch1 := make(chan string, 10)
	ch2 := make(chan string, 10)
	go func() {
		for {
			send("ch1", ch1)
		}
	}()
	go func() {
		for {
			send("ch2", ch2)
		}
	}()
	for {
		select {
		case <-ch1:
			fmt.Println("data recv on ch1 : ", <-ch1)
		case <-ch2:
			fmt.Println("data recv on ch2 : ", <-ch2)
		}
	}

}
