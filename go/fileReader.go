// read from file

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)
func main() {
	f, err := os.Open("test.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		fmt.Println(line)
	}

	
}

