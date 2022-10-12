package main

import "fmt"

type Person struct {
	age    int
	name   string
	salary float64
}

type PersonOptionFunc func(*Person)

func WithName(name string) PersonOptionFunc {
	return func(p *Person) {
		p.name = name
	}
}

func main() {
	p := &Person{}
	WithName("Anton")(p)
	fmt.Println(p)
}
