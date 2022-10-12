package main

import (
	"crypto/rand"
	"fmt"
	"log"
)

func uuid() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}

func djb2(s string) uint64 {
	var hash uint64 = 5381

	for _, c := range s {
		hash = ((hash << 5) + hash) + uint64(c)
		// the above line is an optimized version of the following line:
		//hash = hash * 33 + uint64(c)
		// which is easier to read and understand...
	}

	return hash
}

func main() {
	for i := 0; i < 100; i++ {
		a := djb2(uuid())
		print(a)
		print("\n")
	}

}
