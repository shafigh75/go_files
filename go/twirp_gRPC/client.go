package main

import (
	"context"
	"fmt"
	"mohammad/twirp"
	"net/http"
	"os"
)

func main() {
	client := twirp.NewHaberdasherProtobufClient("http://localhost:8080", &http.Client{})

	hat, err := client.MakeHat(context.Background(), &twirp.Size{Inches: 12})
	if err != nil {
		fmt.Printf("oh no: %v", err)
		os.Exit(1)
	}
	fmt.Printf("I have a nice new hat: %+v", hat)
}
