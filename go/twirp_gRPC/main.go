package main

import (
	"log"
	"net/http"

	"mohammad/server"
	"mohammad/twirp"
)

func main() {
	server := &server.Server{} // implements Haberdasher interface
	twirpHandler := twirp.NewHaberdasherServer(server)
	log.Println("server started on port 8080")

	http.ListenAndServe(":8080", twirpHandler)
}
