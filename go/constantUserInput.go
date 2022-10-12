package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

func main () {
	//var input string
	ch := make(chan string)
	reader := bufio.NewReader(os.Stdin)
	var wg sync.WaitGroup
	for {
		wg.Add(1)
		go func(){
			fmt.Println("enter your message: ")
			input, _ := reader.ReadString('\n')
			send(ch,input)
			wg.Done()
		}()
		wg.Add(1)
		go func (){
			recv(ch)
			wg.Done()
		}()
		wg.Wait()
	}
	
}


func send(ch chan string,str string){
	ch <- str

	
}


func recv(ch chan string) {
	fmt.Println("you enterd: ",<-ch)

	
}
