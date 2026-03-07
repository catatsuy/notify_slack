package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"time"
)

func init() {
	flag.Parse()
}

func main() {
	for i := range 1000 {
		fmt.Printf("Welcome %d times\n", i)
		// sleep 10ms-30ms
		time.Sleep((time.Duration)(rand.IntN(3)+1) * 10 * time.Millisecond)
	}
}
