package main

import (
    "fmt"
    "strings"

    "github.com/chzyer/readline"
    "github.com/spf13/cobra"
)

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
        readline.PcItem("greet", readline.PcItem("name")),
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
        if strings.Contains(line, "greet") {
          args := strings.Fields(line)
          if len(args) > 1{
            fmt.Printf("Hello %s\n", args[1])
            line = "bypass"
          }
        }

        handleCommand(line)
    }
}

func handleCommand(input string) {
    switch input {
    case "help":
        fmt.Println("Available commands: help, exit, greet [name]")
    case "greet":
        fmt.Println("Usage: greet [name]")
    case "bypass":
    default:
        fmt.Printf("Unknown command: %s\n", input)
    }
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        return
    }
}
