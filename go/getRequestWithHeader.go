// You can edit this code!
// Click here and start typing.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"
	"net/http"
	"sync"
)

func req(url string) string {
	 client := &http.Client{
        	Timeout: time.Second * 10,
    }
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhcHAiOiIzNSIsImF1ZCI6IjI3b2ljcGF0bnZ3Nnk3cDNmbWFqcWxzeiIsImV4cCI6IjIwMjEtMDctMjBUMDU6NTc6MjUuMTQ1MzU3NjI5WiIsImlkIjoiIiwicm9sZSI6bnVsbH0.l9Rah5oWXqzCs_g1ggT_A91dM5UMC9pmcfSZ80wzBxs")
	if err != nil {
		log.Fatalln(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("ERROR")
	}
        defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	//Convert the body to type string
	sb := string(body)
	return sb

}

func main() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		res := req("https://favagateway.ir/new/v1/ticket/1000123872061791")
		fmt.Println(res)
		wg.Done()
	}()
	wg.Wait()
}

