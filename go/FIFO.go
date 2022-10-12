package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	a := make(chan int, 1000)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		send(a, 1)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		send(a, 2)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		send(a, 3)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		send(a, 4)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		send(a, 5)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		send(a, 6)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		recv(a)
		wg.Done()
	}()

	go func() {
		for {
		getSize(a)
		time.Sleep(time.Second*3)
	}
	}()
	wg.Wait()
	fmt.Println("DONE!")
}

func send(ch chan int, num int) {
	for {
		time.Sleep(time.Second / 5)
		ch <- num
		fmt.Println("number inserted :  ", num)
	}

}

func recv(ch chan int) {
	for {
		time.Sleep(time.Second *2 )
		fmt.Println("output", <-ch)
	}
}


func getSize(ch chan int) {
		size := len(ch)
		fmt.Println("_________________________________________")
		fmt.Println("\nsize of buffer:   ",size)
		fmt.Println("_________________________________________")
	}
