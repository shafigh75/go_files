package main

import (
    "fmt"
    "log"
    "os"
    "github.com/gocolly/colly"
    "github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Name:  "excuse",
        Usage: "make an excuse for your sorry ass Bro!",
	UsageText: "excuse [Command]",
        Action: func(*cli.Context) error {
	    crawl()
            return nil
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}

func crawl() {
  c := colly.NewCollector(
  )
  c.OnHTML("a", func(e *colly.HTMLElement) {
	  fmt.Printf("%s",e.Text)
  })
  c.Visit("http://programmingexcuses.com/")
}


