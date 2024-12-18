// TEST:
// curl -X POST -H "Content-Type: application/json" -d '{"key": "foo", "value": "bar"}' http://localhost:8080/set
// curl -X GET "http://localhost:8080/get?key=foo"
// curl -X DELETE "http://localhost:8080/delete?key=foo"


package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
)

// InMemoryStore represents a simple in-memory key-value store.
type InMemoryStore struct {
    mu    sync.RWMutex
    store map[string]string
}

// NewInMemoryStore creates a new instance of InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
    return &InMemoryStore{
        store: make(map[string]string),
    }
}

// Set adds a key-value pair to the store.
func (s *InMemoryStore) Set(key, value string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.store[key] = value
}

// Get retrieves a value by key from the store.
func (s *InMemoryStore) Get(key string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    value, exists := s.store[key]
    return value, exists
}

// Delete removes a key-value pair from the store.
func (s *InMemoryStore) Delete(key string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.store, key)
}

// APIResponse represents a standard API response.
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// HTTP handlers
func (store *InMemoryStore) setHandler(w http.ResponseWriter, r *http.Request) {
    var req map[string]string
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    key, value := req["key"], req["value"]
    store.Set(key, value)
    json.NewEncoder(w).Encode(APIResponse{Success: true})
}

func (store *InMemoryStore) getHandler(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    if value, exists := store.Get(key); exists {
        json.NewEncoder(w).Encode(APIResponse{Success: true, Data: value})
    } else {
        json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Key not found"})
    }
}

func (store *InMemoryStore) deleteHandler(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    store.Delete(key)
    json.NewEncoder(w).Encode(APIResponse{Success: true})
}

func main() {
    store := NewInMemoryStore()

    http.HandleFunc("/set", store.setHandler)
    http.HandleFunc("/get", store.getHandler)
    http.HandleFunc("/delete", store.deleteHandler)

    fmt.Println("Starting server on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Println("Error starting server:", err)
    }
}
