package main

import (
	"fmt"
	"sync"
)

func main() {
	wg := &sync.WaitGroup{}
	ch := make(chan int)
	
	wg.Add(2)

	go func (ch chan int, wg *sync.WaitGroup)  {
		fmt.Println(<-ch)
		close(ch)
		// Recevies zero value for channel type as channel is closed.
		fmt.Println(<-ch)
		wg.Done()
	}(ch, wg)

	go func (ch chan int, wg *sync.WaitGroup)  {
		ch <- 42
		close(ch)
		// Panics: sending over closed channel
		ch <- 27
		wg.Done()
	}(ch, wg)
	wg.Wait()
}