package main

import (
    "fmt"
    "net/http"
    "sync"
)

var (
    clients   = make(map[chan string]bool)
    clientsMu sync.Mutex
)

func main() {
    http.HandleFunc("/", serveIndex) // Serve the HTML file
    http.HandleFunc("/events", eventsHandler)
    http.HandleFunc("/send", sendHandler)

    fmt.Println("Server is running on :8080")
    http.ListenAndServe(":8080", nil)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "index.html") // Serve the index.html file
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
    // Set headers for SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // Create a channel for the client
    messageChan := make(chan string)

    // Register the client
    clientsMu.Lock()
    clients[messageChan] = true
    clientsMu.Unlock()

    // Send messages to the client
    defer func() {
        clientsMu.Lock()
        delete(clients, messageChan)
        clientsMu.Unlock()
        close(messageChan)
    }()

    for msg := range messageChan {
        fmt.Fprintf(w, "data: %s\n\n", msg)
        flusher, ok := w.(http.Flusher)
        if ok {
            flusher.Flush()
        }
    }
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    message := r.FormValue("message")

    // Broadcast the message to all clients
    clientsMu.Lock()
    for client := range clients {
        client <- message
    }
    clientsMu.Unlock()

    w.WriteHeader(http.StatusOK)
}
