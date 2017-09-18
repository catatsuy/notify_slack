package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"
)

func init() {
	flag.Parse()
	rand.Seed(time.Now().Unix())
}

func main() {
	for i := 0; i < 1000; i++ {
		fmt.Printf("Welcome %d times\n", i)
		// sleep 10ms-30ms
		time.Sleep((time.Duration)(rand.Intn(3)+1) * 10 * time.Millisecond)
	}
}
