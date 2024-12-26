package main

import (
    "fmt"
    "net/rpc"
    "strings"

    "github.com/chzyer/readline"
    "github.com/spf13/cobra"
)

// RPCRequest and RPCResponse structures
type RPCRequest struct {
    Key   string `json:"key"`
    Value string `json:"value,omitempty"`
    TTL   int64  `json:"ttl"` // TTL in seconds
}

type RPCResponse struct {
    Success bool   `json:"success"`
    Data    string `json:"data,omitempty"`
    Error   string `json:"error,omitempty"`
}

var client *rpc.Client

var rootCmd = &cobra.Command{
    Use:   "mycli",
    Short: "My CLI application",
    Long:  `This is a sample CLI application using Cobra with autocompletion and REPL.`,
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Welcome to My CLI! Type 'help' for available commands.")
        startREPL()
    },
}

func startREPL() {
    // Define completer
    completer := readline.NewPrefixCompleter(
        readline.PcItem("help"),
        readline.PcItem("exit"),
        readline.PcItem("set", readline.PcItem("key"), readline.PcItem("value"), readline.PcItem("ttl")),
        readline.PcItem("get", readline.PcItem("key")),
        readline.PcItem("delete", readline.PcItem("key")),
    )

    // Create readline instance
    rl, err := readline.NewEx(&readline.Config{
        Prompt:          "> ",
        HistoryFile:     "/tmp/readline.tmp",
        AutoComplete:    completer,
        InterruptPrompt: "^C",
        EOFPrompt:      "exit",
    })
    if err != nil {
        fmt.Println("Error creating readline:", err)
        return
    }
    defer rl.Close()

    for {
        line, err := rl.Readline()
        if err != nil {
            break
        }
        line = strings.TrimSpace(line)

        if line == "exit" {
            fmt.Println("Exiting REPL.")
            break
        }

        handleCommand(line)
    }
}

func handleCommand(input string) {
    args := strings.Fields(input)
    if len(args) == 0 {
        return
    }

    switch args[0] {
    case "help":
        fmt.Println("Available commands: help, exit, set [key] [value] [ttl], get [key], delete [key]")
    case "set":
        if len(args) < 4 {
            fmt.Println("Usage: set [key] [value] [ttl]")
            return
        }
        key := args[1]
        ttl := int64(0)
        // Get the TTL from the last argument
        fmt.Sscanf(args[len(args)-1], "%d", &ttl)
        // Join all arguments except the first and last as the value
        value := strings.Join(args[2:len(args)-1], " ")
        setKey(key, value, ttl)
    case "get":
        if len(args) != 2 {
            fmt.Println("Usage: get [key]")
            return
        }
        getKey(args[1])
    case "delete":
        if len(args) != 2 {
            fmt.Println("Usage: delete [key]")
            return
        }
        deleteKey(args[1])
    default:
        fmt.Printf("Unknown command: %s\n", input)
    }
}

func setKey(key, value string, ttl int64) {
    req := RPCRequest{Key: key, Value: value, TTL: ttl}
    var resp RPCResponse
    err := client.Call("InMemoryStore.RPCSet", &req, &resp)
    if err != nil {
        fmt.Println("Error calling RPCSet:", err)
        return
    }
    if resp.Success {
        fmt.Println("Key set successfully.")
    } else {
        fmt.Println("Error:", resp.Error)
    }
}

func getKey(key string) {
    req := RPCRequest{Key: key}
    var resp RPCResponse
    err := client.Call("InMemoryStore.RPCGet", &req, &resp)
    if err != nil {
        fmt.Println("Error calling RPCGet:", err)
        return
    }
    if resp.Success {
        fmt.Printf("Value: %s\n", resp.Data)
    } else {
        fmt.Println("Error:", resp.Error)
    }
}

func deleteKey(key string) {
    req := RPCRequest{Key: key}
    var resp RPCResponse
    err := client.Call("InMemoryStore.RPCDelete", &req, &resp)
    if err != nil {
        fmt.Println("Error calling RPCDelete:", err)
        return
    }
    if resp.Success {
        fmt.Println("Key deleted successfully.")
    } else {
        fmt.Println("Error:", resp.Error)
    }
}

func main() {
    // Connect to the RPC server
    var err error
    client, err = rpc.Dial("tcp", "localhost:1234")
    if err != nil {
        fmt.Println("Error connecting to RPC server:", err)
        return
    }
    defer client.Close()

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        return
    }
}
