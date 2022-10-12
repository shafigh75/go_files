package main

import (
    "database/sql"
    "fmt"
    "log"

    _ "github.com/go-sql-driver/mysql"
)

type Person struct {
    Id         int
    Name       string
    Sex 	int
}

func main() {

    db, err := sql.Open("mysql", "mohammad:9090kosenanat@tcp(127.0.0.1:3306)/test")
    defer db.Close()

    if err != nil {
        log.Fatal(err)
    }

    res, err := db.Query("SELECT * FROM person")

    defer res.Close()

    if err != nil {
        log.Fatal(err)
    }

    for res.Next() {

        var person Person
        err := res.Scan(&person.Id, &person.Name, &person.Sex)

        if err != nil {
            log.Fatal(err)
        }

        fmt.Printf("%v\n", person)
    }
}
