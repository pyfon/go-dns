package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const NUM_RECS = 1_000_000
const THREADS = 8
const STRLEN = 12

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func randSeq() string {
	n := STRLEN
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GenRec(ch chan<- string) {
	ch <- fmt.Sprintf("%s\tTXT\t\"%s\"\n", randSeq(), randSeq())
}

func main() {
	fmt.Printf("zone %s.example.com\nttl 300\n\n", randSeq())

	ch := make(chan string)
	var wg sync.WaitGroup

	// Start goroutines
	for i := 0; i < THREADS; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for x := 0; x < NUM_RECS/THREADS; x++ {
				GenRec(ch)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for s := range ch {
		fmt.Printf("%s", s)
	}
}
