// download image from web and write to file

package main


import (
	"fmt"
	"io"
	"net/http"
	"os"
)


func main() {
	url := "http://www.google.com/images/srpr/logo3w.png"
	file, err := os.Create("google.png")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	// copy the response body into the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Image downloaded successfully.")
}
