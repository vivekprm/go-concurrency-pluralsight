package main

import "fmt"

func main() {
	evenCh := make(chan int)
	oddCh := make(chan int)

	go printNumber(evenCh, oddCh)

	for {
		select {
		case x, ok := <-evenCh:
			if ok {
				fmt.Println(x)
			} else {
				evenCh = nil
			}
		case y, ok := <-oddCh:
			if ok {
				fmt.Println(y)
			} else {
				oddCh = nil
			}
		}
		if evenCh == nil && oddCh == nil {
			break
		}
	}
}

func printNumber(evenCh chan int, oddCh chan int) {
	i := 0
	for {
		if i == 20 {
			close(evenCh)
			close(oddCh)
			break
		}
		if i%2 == 0 {
			evenCh <- i
		} else {
			oddCh <- i
		}
		i++
	}
}
