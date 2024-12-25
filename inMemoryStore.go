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





//______________________________________________________________________VERSION 2 _________________________________________________________
//
//
//    THIS CODE IS USING TTL TO SET EXPIRATION FOR THE KEYS:
//    curl -X POST -H "Content-Type: application/json" -d '{"key": "foo", "value": "bar", "ttl": 10}' http://localhost:8080/set
//    curl -X GET "http://localhost:8080/get?key=foo"
//    curl -X DELETE "http://localhost:8080/delete?key=foo"
//
// _________________________________________________________________________________________________________________________________________

package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

// ValueWithTTL represents a value with its expiration time.
type ValueWithTTL struct {
    Value      string
    Expiration int64 // Unix timestamp in seconds
}

// InMemoryStore represents a simple in-memory key-value store with TTL.
type InMemoryStore struct {
    mu    sync.RWMutex
    store map[string]ValueWithTTL
}

// NewInMemoryStore creates a new instance of InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
    return &InMemoryStore{
        store: make(map[string]ValueWithTTL),
    }
}

// Set adds a key-value pair to the store with an optional TTL.
func (s *InMemoryStore) Set(key, value string, ttl int64) {
    s.mu.Lock()
    defer s.mu.Unlock()
    expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
    s.store[key] = ValueWithTTL{Value: value, Expiration: expiration}
}

// Get retrieves a value by key from the store, checking for expiration.
func (s *InMemoryStore) Get(key string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    valueWithTTL, exists := s.store[key]
    if !exists || (valueWithTTL.Expiration > 0 && time.Now().Unix() > valueWithTTL.Expiration) {
        return "", false
    }
    return valueWithTTL.Value, true
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
    var req struct {
        Key   string `json:"key"`
        Value string `json:"value"`
        TTL   int64  `json:"ttl"` // TTL in seconds
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    store.Set(req.Key, req.Value, req.TTL)
    json.NewEncoder(w).Encode(APIResponse{Success: true})
}

func (store *InMemoryStore) getHandler(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    if value, exists := store.Get(key); exists {
        json.NewEncoder(w).Encode(APIResponse{Success: true, Data: value})
    } else {
        json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Key not found or expired"})
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




// _____________________________________________________VERSION 3 _________________________________________________________________
//
//    ADDED TTL AND ALSO THE AUTOMATIC DELETION OF EXPIRED KEYS IN AN EFFICIENT MANNER 
//    curl -X POST -H "Content-Type: application/json" -d '{"key": "foo", "value": "bar", "ttl": 5}' http://localhost:8080/set
//    curl -X GET "http://localhost:8080/get?key=foo"
//    curl -X DELETE "http://localhost:8080/delete?key=foo"
//
// ________________________________________________________________________________________________________________________________



package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

// ValueWithTTL represents a value with its expiration time.
type ValueWithTTL struct {
    Value      string
    Expiration int64 // Unix timestamp in seconds
}

// InMemoryStore represents a simple in-memory key-value store with TTL.
type InMemoryStore struct {
    mu    sync.RWMutex
    store map[string]ValueWithTTL
}

// NewInMemoryStore creates a new instance of InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
    return &InMemoryStore{
        store: make(map[string]ValueWithTTL),
    }
}

// Set adds a key-value pair to the store with an optional TTL.
func (s *InMemoryStore) Set(key, value string, ttl int64) {
    s.mu.Lock()
    defer s.mu.Unlock()
    expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
    s.store[key] = ValueWithTTL{Value: value, Expiration: expiration}
}

// Get retrieves a value by key from the store, checking for expiration.
func (s *InMemoryStore) Get(key string) (string, bool) {
    s.mu.RLock()
    valueWithTTL, exists := s.store[key]
    s.mu.RUnlock() // Unlock before potentially deleting

    if !exists || (valueWithTTL.Expiration > 0 && time.Now().Unix() > valueWithTTL.Expiration) {
        // If the key does not exist or has expired, attempt to delete it
        if exists {
            s.Delete(key) // Delete the expired key
        }
        return "", false
    }

    return valueWithTTL.Value, true
}

// Delete removes a key-value pair from the store.
func (s *InMemoryStore) Delete(key string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.store, key)
}

// Cleanup removes expired keys from the store concurrently.
func (s *InMemoryStore) Cleanup() {
    s.mu.Lock()
    defer s.mu.Unlock()
    now := time.Now().Unix()
    for key, valueWithTTL := range s.store {
        if valueWithTTL.Expiration > 0 && now > valueWithTTL.Expiration {
            delete(s.store, key)
        }
    }
}

// StartCleanupRoutine starts a background goroutine to periodically clean up expired keys.
func (s *InMemoryStore) StartCleanupRoutine(interval time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            <-ticker.C
            s.Cleanup() // Perform cleanup at regular intervals
        }
    }()
}

// APIResponse represents a standard API response.
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// HTTP handlers
func (store *InMemoryStore) setHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Key   string `json:"key"`
        Value string `json:"value"`
        TTL   int64  `json:"ttl"` // TTL in seconds
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    store.Set(req.Key, req.Value, req.TTL)
    json.NewEncoder(w).Encode(APIResponse{Success: true})
}

func (store *InMemoryStore) getHandler(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    if value, exists := store.Get(key); exists {
        json.NewEncoder(w).Encode(APIResponse{Success: true, Data: value})
    } else {
        json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Key not found or expired"})
    }
}

func (store *InMemoryStore) deleteHandler(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    store.Delete(key)
    json.NewEncoder(w).Encode(APIResponse{Success: true})
}

func main() {
    store := NewInMemoryStore()

    // Start the cleanup routine every 10 seconds
    store.StartCleanupRoutine(10 * time.Second)

    http.HandleFunc("/set", store.setHandler)
    http.HandleFunc("/get", store.getHandler)
    http.HandleFunc("/delete", store.deleteHandler)

    fmt.Println("Starting server on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Println("Error starting server:", err)
    }
}
