# Why is concurrency important?
AMD Ryzen ThreadRipper PRO 5955WX
- Single threaded rating 3282
- Multi threaded rating 97132

If we look at the stats of more powerful CPUs available today, e.g. above AMD Ryzen Threadripper, it's single threaded processing is pretty good. It's 3282. However, if you are able to take advantage of the entire processor, it's multi threaded rating id 97132. 30 times of performance is available to us if we can use all the cores available on that CPU.

INTEL Xeon Platinum 8380
- single-threaded rating 2385
- multi-threaded rating 62318

We see similar difference in case of Intel CPUs.

https://www.cpubenchmark.net/high_end_cpus.html

So, concurrency allows us to get the most out of the infrastructure that our applications are running on. 

So why GO? Why is Go such a popular choice in this world of concurrent development?
Go was designed with Concurrency in Mind. 

What do we mean by that?
We have concepts such as:
- Goroutines 
  - Which allows us to kick off concurrent tasks easily.
- Channels
  - Allow these Goroutines to communicate with each other safely and efficiently.
- Sync Package
  - Part of the Go standard library, that has several components and several bits of functionality that further simplify the development of concurrent programs.
- Communicating Sequential Processes (CSP)
  - Finally, at the heart of it all, Go's concurrency is built on the model of communicating sequential processes which has proven itself to be one of the most reliable and predictable models for concurrent development.

## Prerequisites
Go Fundamental course

We are going to model a simple inventory management system.
- Take Orders
- Move Orders using Order Processing Pipeline

This is very common workflow that is used in Concurrent programs, where we have different tasks that are working independently, but they are doing some portion of a greater amount of work.
So for example in an order processing pipeline we might have: 
- One worker or one go routine that's receiving orders in from whatever our endpoints are, may be we got an ecommerce site that's receiving these orders.
- One worker that's going to validate those orders to make sure that they're correct.
- Eventually we might add another order that's going to reserve inventory and then finally we might have a worker that's going to fulfill that inventory and send those orders to the people that ordered them.

order.go
```go
package main

import "fmt"

type orderStatus int

const (
	none orderStatus = iota
	new
	received
	reserved
	filled
)

var orders = []order{}

type order struct {
	ProductCode int
	Quantity    float64
	Status      orderStatus
}

func (o order) String() string {
	return fmt.Sprintf("Product code: %v, Quantity: %v, Status: %v \n",
		o.ProductCode, o.Quantity, orderStatusToText(o.Status))
}

func orderStatusToText(o orderStatus) string {
	switch o {
	case none:
		return "none"
	case new:
		return "new"
	case received:
		return "received"
	case reserved:
		return "reserved"
	case filled:
		return "filled"
	default:
		return "unknown status"
	}
}
```

Go has Stringer interface which has String() method, which we can implement to print our structs.

main.go
```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
)

func main() {
	receiveOrders()
	fmt.Println(orders)
}

func receiveOrders() {
	for _, rawOrder := range rawOrders {
		var newOrder order
		err := json.Unmarshal([]byte(rawOrder), &newOrder)
		if err != nil {
			log.Println(err)
			continue
		}
		orders = append(orders, newOrder)
	}
}

var rawOrders = []string{
	`{"productCode": 1111, "quantity": 5, "status": 1}`,
	`{"productCode": 2222, "quantity": 42.3, "status": 1}`,
	`{"productCode": 3333, "quantity": 19, "status": 1}`,
	`{"productCode": 4444, "quantity": 8, "status": 1}`,
}
```

So above program works just fine, we receive order and print them. But we don't want to do this synchronously. We want to do this asynchronously. 

So we are going to modify our main function to receive orders as asynchronous task.
We can change receive order to a goroutine as below:
```go
go receiveOrders()
```

but we want to wait for this goroutine to finish and for that we use waitgroup.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)
	go receiveOrders(&wg)
	wg.Wait()
	fmt.Println(orders)
}

func receiveOrders(wg *sync.WaitGroup) {
	for _, rawOrder := range rawOrders {
		var newOrder order
		err := json.Unmarshal([]byte(rawOrder), &newOrder)
		if err != nil {
			log.Println(err)
			continue
		}
		orders = append(orders, newOrder)
	}
	wg.Done()
}

var rawOrders = []string{
	`{"productCode": 1111, "quantity": 5, "status": 1}`,
	`{"productCode": 2222, "quantity": 42.3, "status": 1}`,
	`{"productCode": 3333, "quantity": 19, "status": 1}`,
	`{"productCode": 4444, "quantity": 8, "status": 1}`,
}
```

So it gets the same result as running synchronously, but we are now working concurrently. So if the main function had something else to do then receiveOrders() would continue to process and not block the main function.

Now we are going to add another function that's going to validate the orders that we receive and that's going to receive the orders from this receiveOrders function using channels.
We are going to have very simple validation, we are just going to check quantity of the order and make sure we always have a positive value.

So instead of appending order to orders slice we are going to send the order to order channel as soon as we are done unmarshalling that order.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	var receivedOrderCh = make(chan order)
	var validOrderCh = make(chan order)
	var invalidOrderCh = make(chan invalidOrder)
	go receiveOrders(receivedOrderCh)
	go validateOrders(receivedOrderCh, validOrderCh, invalidOrderCh)
	wg.Add(1)
	go func() {
		order := <-validOrderCh
		fmt.Printf("Valid order received: %v\n", order)
		wg.Done()
	}()
	go func() {
		invalidOrder := <-invalidOrderCh
		fmt.Printf("InValid order received: %v, issue: %v\n", invalidOrder.order, invalidOrder.err)
		wg.Done()
	}()
	wg.Wait()
}

func receiveOrders(out chan order) {
	for _, rawOrder := range rawOrders {
		var newOrder order
		err := json.Unmarshal([]byte(rawOrder), &newOrder)
		if err != nil {
			log.Println(err)
			continue
		}
		out <- newOrder
	}
}

func validateOrders(in, out chan order, errch chan invalidOrder) {
	order := <-in
	if order.Quantity <= 0 {
		errch <- invalidOrder{
			order: order,
			err:   fmt.Errorf("Quantity must be positive %d", order.Quantity),
		}
	} else {
		out <- order
	}
}

var rawOrders = []string{
	`{"productCode": 1111, "quantity": 5, "status": 1}`,
	`{"productCode": 2222, "quantity": 42.3, "status": 1}`,
	`{"productCode": 3333, "quantity": 19, "status": 1}`,
	`{"productCode": 4444, "quantity": 8, "status": 1}`,
}
```
This is going to work with only on workgroup add call with the index 1 because channels are blocking. So when we process all these orders in the receiveOrders goroutine, that goroutine is not going to stop until it's channel is drained. What's is draining that channel is validateOrders goroutine. But it's going to block until it's channel are drained. Which are the error channel and out channel. Only one is going to get the message the way that our scenario is written. So those are going to block, when is that going to be cleared?

That's going to be cleared with one of anonymous goroutines. So all of our goroutines are going to block until we have the message come out the other side. So everything is going to naturally wait because channels will cause our application to wait until we get through the entire workflow. So we only need that one Done() call.

We receive one order only because we don't have a loop.

------

# Goroutines
Where the Goroutine actually sits in your go program?
- Go program interacts with Go Runtime and specific part of the runtime we want to focus on is something called the **scheduler**. And finally at our application stack is the operating system that the application is going to interact with.
- Go program sits between Go program and Go runtime scheduler. 
- Between the Scheduler and Operating system, we have a construct called **Threads**.
- Scheduler takes these Goroutines, also known as virtual threads, these are maintained purely within our Go program.
- A Goroutine is limited. Since it's not directly with the operating system, it can't do anything on its own. That's where scheduler comes in. The scheduler will map goroutines onto the operating system threads. In order fot goroutines to do anything it has to be scheduled onto a thread.
- Now one of the nice things about the architecture that Go offers us with this model is the scheduler can change things up.
- Scheduler can decide based on what those Goroutines are doing, it can optimize the amount of time that the Goroutines are spending on threads to improve the performance of our program as much as possible.
- That's one benefit of the Goroutine. It allows our Go program to have more control over what is actually working concurrently or not.

## Thread vs Goroutines
- **Threads** have its own execution stack and that stack has its own memory that it can work with. **Goroutines** have their own execution stack as well.
- A **Thread** typically has a fixed stack space, around 1MB. So when operating system creates the thread it allocates approximately 1 MB of memory to it. If you have heard about StackOverflow, well a stack overflow often occurs when this 1 MB of stack space has been exceeded. With **Goroutines** they've got variable stack space (starts @2KB). Which is 500 times smaller than the typical thread that you're going to create in your operating system. Why is that a benifit?
  - The first thing that that does is that makes goroutines extremely lightweight in terms of memory footprint. So we could have 500 goroutines and still occupy just the same amount of memory as an operating system thread and then Go scheduler would then make sure that those 500 goroutines all get equal share of time. Additionally since the stack space is variable, Go can actually Go beyond the 1 MB limit. The maximum size of a Goroutine stack is somewhere about 2 GB, so large enough for almost any application usage.
- **Threads** are managed by the OS. Goroutine is managed by the Go runtime specifically that scheduler. Benifit is actually going to go towards threads. Working directly with threads is going to be slightly more performant for  given concurrency model. So if you got 4 threads in your program and you have got four Goroutines in your program generally the threaded model is going to be slightly more efficient because the scheduler does have some overhead.
- **Threads** are relatively expensive because in order for your program to create a thread it actually has to interact with the operating system, and the operating system has to create that thread and return it to your program. It's relatively expensive operation in terms of the things that your program might also be doing. Whereas the **Goroutine** they live directly within the Go runtime. So they don't have to interact with the operating system at all and that once again leads to a relatively lightweight and very very performant model for constructing goroutines.

It makes very very easy to run very high level of concurrency in Go and maintain a very efficient execution. With Goroutines we can have thousands of Goroutines active in our program at one time, and the scheduler is going to make sure that works as efficiently as possible. So the level of concurrency can be higher, making it more likely that we're taking full advantage of the resources on the system that we're running on.

## Lifecycle of a Goroutine

## Advice Regarding Goroutines

# Channels

## Buffered & Unbuffered Channels
Unbuffered Channel
```go
var ch = make(chan string)
func sender() {
    ch <- "message"
}
func receive() {
    msg := <- ch
    fmt.Println(msg)
}
func main() {
    // synchronization elided for clarity
    go sender()
    go receive()
}
```

Buffered Channel
```go
var ch = make(chan string, 1)
func sender() {
    ch <- "message"
}
func receive() {
    msg := <- ch
    fmt.Println(msg)
}
func main() {
    // synchronization elided for clarity
    go sender()
    go receive()
}
```

This decouples the sending side of the channel from receiving side of the channel, and that's one of the primary usecases for buffered channels.

```go
func main() {
	ch := make(chan string)
	ch <- "message"
	fmt.Println(<-ch)
}
```

it causes deadlock as it blocks at line 2 of the main. changing it buffered channel works.

```go
func main() {
	ch := make(chan string, 1)
	ch <- "message"
	fmt.Println(<-ch)
}
```

## Directional Channels
By default when we construct a channel in Go it can do both, receive a message and send a meesage. This is what is called bi-directional channel. However, they don't provide the level of clarity that we want.

```go
func main() {
    ch := make(chan string)
    go func(ch chan string) {
        ch <- "message"
    } (ch)
    go func(ch chan string) {
        fmt.Println(<-ch)
    } (ch)
}
```

In above example, one go routing is trying to send a essage and the other one just receiving the message from the channel. Which is often the case, since we are passing bidirectional channel we can do otherwse and increase the chaneces of bug and clarity as we get the intent.

```go
func main() {
    ch := make(chan string)
    go func(ch chan<- string) {
        ch <- "message"
    } (ch)
    go func(ch <-chan string) {
        fmt.Println(<-ch)
    } (ch)
}
```

It's very important best practice to use.
We can modify the earlier program as below:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	var receivedOrderCh = make(chan order)
	var validOrderCh = make(chan order)
	var invalidOrderCh = make(chan invalidOrder)
	go receiveOrders(receivedOrderCh)
	go validateOrders(receivedOrderCh, validOrderCh, invalidOrderCh)
	wg.Add(1)
	go func(validOrderCh <-chan order) {
		order := <-validOrderCh
		fmt.Printf("Valid order received: %v\n", order)
		wg.Done()
	}(validOrderCh)
	go func(invalidOrderCh <-chan invalidOrder) {
		invalidOrder := <-invalidOrderCh
		fmt.Printf("InValid order received: %v, issue: %v\n", invalidOrder.order, invalidOrder.err)
		wg.Done()
	}(invalidOrderCh)
	wg.Wait()
}

func receiveOrders(out chan<- order) {
	for _, rawOrder := range rawOrders {
		var newOrder order
		err := json.Unmarshal([]byte(rawOrder), &newOrder)
		if err != nil {
			log.Println(err)
			continue
		}
		out <- newOrder
	}
}

func validateOrders(in <-chan order, out chan<- order, errch chan<- invalidOrder) {
	order := <-in
	if order.Quantity <= 0 {
		errch <- invalidOrder{
			order: order,
			err:   fmt.Errorf("Quantity must be positive %d", order.Quantity),
		}
	} else {
		out <- order
	}
}

var rawOrders = []string{
	`{"productCode": 1111, "quantity": 5, "status": 1}`,
	`{"productCode": 2222, "quantity": 42.3, "status": 1}`,
	`{"productCode": 3333, "quantity": 19, "status": 1}`,
	`{"productCode": 4444, "quantity": 8, "status": 1}`,
}
```

It gives more clarity and get compile time error if we try to send on recieve only channel for example.

## Control Flow With Channels
### Select Statements
```go
func main() {
    ch1 := make(chan int, 1)
    ch2 := make(chan string, 1)

    ch1 <- 999
    select {
        case msg := <- ch1:
            fmt.Println(msg)
        case msg := <- ch2:
            fmt.Println(msg)
    }
}
```

If we send on ch2 as well we get undefined behaviour and will see random outputs.
```go
func main() {
    ch1 := make(chan int, 1)
    ch2 := make(chan string, 1)

    ch1 <- 999
    ch2 <- "message"
    select {
        case msg := <- ch1:
            fmt.Println(msg)
        case msg := <- ch2:
            fmt.Println(msg)
    }
}
```

If nothing is send we will be blocked and deadlocked.

```go
func main() {
    ch1 := make(chan int, 1)
    ch2 := make(chan string, 1)

    select {
        case msg := <- ch1:
            fmt.Println(msg)
        case msg := <- ch2:
            fmt.Println(msg)
    }
}
```

So it blocks until one case is selected. To fix this we can have default which executes if none of the cases can be selected.

```go
func main() {
    ch1 := make(chan int, 1)
    ch2 := make(chan string, 1)

    select {
        case msg := <- ch1:
            fmt.Println(msg)
        case msg := <- ch2:
            fmt.Println(msg)
        default:
            fmt.Println("default")
    }
}
```

Let's use this concept in above program, as we only get message either on valid channel or invalid channel and we don't have any default case.

```go
func main() {
	var wg sync.WaitGroup
	var receivedOrderCh = make(chan order)
	var validOrderCh = make(chan order)
	var invalidOrderCh = make(chan invalidOrder)
	go receiveOrders(receivedOrderCh)
	go validateOrders(receivedOrderCh, validOrderCh, invalidOrderCh)
	wg.Add(1)
	go func(validOrderCh <-chan order, invalidOrderCh <-chan invalidOrder) {
		select{
		case order := <-validOrderCh:
			fmt.Printf("Valid order received: %v\n", order)
		case invalidOrder := <-invalidOrderCh:
			fmt.Printf("InValid order received: %v, issue: %v\n", invalidOrder.order, invalidOrder.err)
		}
		wg.Done()
	}(validOrderCh, invalidOrderCh)
	wg.Wait()
}
```

### For loops
```go
func main() {
    ch := make(chan string, 3)
    for _, v := range [...]arr{"foo", "bar", "baz"} {
        ch <- v
    }

    for msg := range ch {
        fmt.Println(msg)
    }
}
```

We will get "foo", "bar", "baz" and then deadlock and the reason is we don't know in channel is done we are waiting to recieve but nothing is being send. 

To fix this we use builtin close function 
```go
func main() {
    ch := make(chan string, 3)
    for _, v := range [...]arr{"foo", "bar", "baz"} {
        ch <- v
    }
    close(Ch)
    for msg := range ch {
        fmt.Println(msg)
    }
}
```

Let's fix our program:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	var receivedOrderCh = make(chan order)
	var validOrderCh = make(chan order)
	var invalidOrderCh = make(chan invalidOrder)
	go receiveOrders(receivedOrderCh)
	go validateOrders(receivedOrderCh, validOrderCh, invalidOrderCh)
	wg.Add(1)
	go func(validOrderCh <-chan order, invalidOrderCh <-chan invalidOrder) {
	loop:
		for {
			select {
			case order, ok := <-validOrderCh:
				if ok {
					fmt.Printf("Valid order received: %v\n", order)
				} else {
					break loop
				}
			case invalidOrder, ok := <-invalidOrderCh:
				if ok {
					fmt.Printf("InValid order received: %v, issue: %v\n", invalidOrder.order, invalidOrder.err)
				} else {
					break loop
				}
			}
		}
		wg.Done()
	}(validOrderCh, invalidOrderCh)
	wg.Wait()
}

func receiveOrders(out chan<- order) {
	for _, rawOrder := range rawOrders {
		var newOrder order
		err := json.Unmarshal([]byte(rawOrder), &newOrder)
		if err != nil {
			log.Println(err)
			continue
		}
		out <- newOrder
	}
	close(out)
}

func validateOrders(in <-chan order, out chan<- order, errch chan<- invalidOrder) {
	for order := range in {
		if order.Quantity <= 0 {
			errch <- invalidOrder{
				order: order,
				err:   fmt.Errorf("quantity must be positive %d", order.Quantity),
			}
		} else {
			out <- order
		}
	}
	close(out)
	close(errch)
}

var rawOrders = []string{
	`{"productCode": 1111, "quantity": 5, "status": 1}`,
	`{"productCode": 2222, "quantity": 42.3, "status": 1}`,
	`{"productCode": 3333, "quantity": 19, "status": 1}`,
	`{"productCode": 4444, "quantity": 8, "status": 1}`,
}
```

The **ok** parameter in the select case tells if message was received from closed channel or not.

## Common Concurrency Patterns
### Non Blocking Error Channel
Let's look at below example:

```go
var (
    in = make(chan string)
    out = make(chan int)
    errCh = make(chan error)
)

func worker(in <-chan string, out chan<- int, errCh chan<- error) {
    for msg := range in {
        i, err := strconv.Atoi(msg)
        if err != nil {
            errCh <- err
            return
        }
        out <- i
    }
}
```

In case of error we are populating errCh and in case of no error, we are populating out channel.
If we remember the earlier concept where we talked that every Goroutine should have an exit strategy. We know how every Goroutine is going to terminate before we start it. 

Here is the question? What if the **error channel** above is never drained?
We can't guarantee that it's going to happen because in this program at this point it's consumer of the Goroutine that has that responsibility. So we have introduced a potential for a resource leak if our worker function is not being consumed properly.

Now the same technically exists for **out** channel, but generally we can be pretty safe in assuming that the out channel is going to be drained because that's probably why this goroutine has been initiated in first place.

So instead we are going to create what's called a non-blocking error channel, and this is generally **one of the only places that we actually recommend using buffer channel**.

```go
var (
    in = make(chan string)
    out = make(chan int)
    errCh = make(chan error, 1)
)

func worker(in <-chan string, out chan<- int, errCh chan<- error) {
    for msg := range in {
        i, err := strconv.Atoi(msg)
        if err != nil {
            errCh <- err
            return
        }
        out <- i
    }
}
```

In this case, the error channel is guaranteed to have a receiver. There's no leaks. So it's reasonable to assume that the **out** channel is going to be drained by the consumer, and it's normal to expect that the error channel is going to be received as well. But if somebody decides not to, if the consumer decides not to that buffer is going to give us a destination for that error to be stored.

Now eventually, if that error channel never is drained then nothing is going to have reference to that channel and the garbage collector is going to cleanup the channel, as well as that channel that's sitting in the buffer.

What that means is that our worker Goroutine does have an exit strategy, even if an error occurs, that error is always going to have a place to go and so we can be guaranteed that we don't have a resource leak from this worker, assuming that the output channel is drained and that's pretty reasonable assumption.

### Encapsulating GoRoutines
Let's look at this example:

```go
var (
    in = make(chan string)
    out = make(chan int)
    errCh = make(chan error)
)

func worker(in <-chan string, out chan<- int, errCh chan<- error) {
    for msg := range in {
        i, err := strconv.Atoi(msg)
        if err != nil {
            errCh <- err
            return
        }
        out <- i
    }
}
```

errorCh and out channels are the output from the worker function, but notice that we are assuming within the worker that we are not receiving **nil** values.
What if the output channels are uninitialized?

Channels in Go can have the value **nil**. An uninitialized channel variable does have a value **nil**. So the question is what happens if we send a message into a nil channel?
The answer is that the application panics. So in wroker function we are relying on the consumer to send us properly initialized channels.

This is commonly done in Go, and it's not a problem, however an alternative that you might want to consider is to **encapsulate the Goroutine**. It looks like below:

```go
var (
    in = make(chan string)
)

func worker(in <-chan string) (chan int, chan error) {
    errCh := make(chan error, 1)
    out := make(chan int)
    go func() {
        for msg := range in {
            i, err := strconv.Atoi(msg)
            if err != nil {
                errCh <- err
                return
            }
            out <- i
        }
    }()
    return out, errCh
}
```

Here we moved out and error channel from function parameter to return value so our worker is not meant to be initialized with those channels. It's meant to be called synchronously becaues it got return values and **return values from Goroutines are always ignored**. So we're intending this worker function to be called synchronously. Within the worker we are going to initialize the channels. And look where we have created the Goroutine.

So our Goroutine is kicked off within this synchronous function, so we get the value of making sure that those channels are always properly initialized. So now what are the benefits and costs of this pattern?

One cost is that consumer no longer has the flexibility of determining the level of concurrency with this Goroutine. The level of concurrency is controlled within the worker function itself because it's responsible for creating the Goroutine.

So If you have a worker that you might want to spawn a variable number of, may be sometimes you want to spawn 1, sometimes you want to spawn 10, then this pattern isn't necessarily going to be the best fit for you. However, if defensiveness is more important to you than controlling the level of concurrency, then you can certainly use this pattern.

There are alternatives, we could certainly get around that by passing in a level of concurrency parameter. Let's apply these into our course demo program and modify **receiveOrder** and **validateOrder** functions to use these patterns..

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	receivedOrderCh := receiveOrders()
	validOrderCh, invalidOrderCh := validateOrders(receivedOrderCh)
	wg.Add(1)
	go func(validOrderCh <-chan order, invalidOrderCh <-chan invalidOrder) {
	loop:
		for {
			select {
			case order, ok := <-validOrderCh:
				if ok {
					fmt.Printf("Valid order received: %v\n", order)
				} else {
					break loop
				}
			case invalidOrder, ok := <-invalidOrderCh:
				if ok {
					fmt.Printf("InValid order received: %v, issue: %v\n", invalidOrder.order, invalidOrder.err)
				} else {
					break loop
				}
			}
		}
		wg.Done()
	}(validOrderCh, invalidOrderCh)
	wg.Wait()
}

func receiveOrders() <-chan order {
	out := make(chan order)
	go func() {
		for _, rawOrder := range rawOrders {
			var newOrder order
			err := json.Unmarshal([]byte(rawOrder), &newOrder)
			if err != nil {
				log.Println(err)
				continue
			}
			out <- newOrder
		}
		close(out)
	}()
	return out
}

func validateOrders(in <-chan order) (<-chan order, <-chan invalidOrder) {
	out := make(chan order)
	errch := make(chan invalidOrder, 1)
	go func() {
		for order := range in {
			if order.Quantity <= 0 {
				errch <- invalidOrder{
					order: order,
					err:   fmt.Errorf("quantity must be positive %f", order.Quantity),
				}
			} else {
				out <- order
			}
		}
		close(out)
		close(errch)
	}()
	return out, errch
}

var rawOrders = []string{
	`{"productCode": 1111, "quantity": 5, "status": 1}`,
	`{"productCode": 2222, "quantity": 42.3, "status": 1}`,
	`{"productCode": 3333, "quantity": 19, "status": 1}`,
	`{"productCode": 4444, "quantity": 8, "status": 1}`,
}
```

Here we can handle one error but not multiple as we are running for loop, if no consumer of error channel is present, we will only be able to handle on error.

## Foundational Concurrency Patterns
There are four foundational concurrency patterns that make up every other pattern we're going to see in Go or in any language.

### Single Producer & Single Consumer
Single producer means there is one Goroutine that is sending messages into a channel. Single consumer mean that there is one goroutine receiving messages from that channel. This is what we saw in our demo example.

Here we add another worker called reserveOrder, that takes valid orders and reserve the order and send over output channel

```go
// messages coming in from valid order channel.
// output orders from receive only channel, as we don't want consumers to push messges 
// into the channel
func reserveInventory(in <-chan order) <-chan order {
	out := make(chan order)
	go func() {
		for o := range in {
			o.Status = reserved
			out <- o
		}
		close(out)
	}()
	return out
}
```

But now we got two consumers of validOrderChannel, we don't need that Goroutine in main.

```go
func main() {
	var wg sync.WaitGroup
	receivedOrderCh := receiveOrders()
	validOrderCh, invalidOrderCh := validateOrders(receivedOrderCh)
	reservedInventoryCh := reserveInventory(validOrderCh)
	wg.Add(2)
	
	go func(invalidOrderCh <-chan invalidOrder) {
		for order := range invalidOrderCh {
			fmt.Printf("Invalid order received: %v, issue: %v", order.order, order.err)
		}
		wg.Done()
	}(invalidOrderCh)

	go func(reservedInventoryCh <-chan order) {
		for order := range reservedInventoryCh {
			fmt.Printf("Inventory reserved for: %v\n", order)
		}
		wg.Done()
	}(reservedInventoryCh)

	wg.Wait()
}
```

### Single Producer & Multiple Consumer
We got multiple Goroutines that are pulling messages from the same channel.

```go
func main() {
	var wg sync.WaitGroup
	receivedOrderCh := receiveOrders()
	validOrderCh, invalidOrderCh := validateOrders(receivedOrderCh)
	reservedInventoryCh := reserveInventory(validOrderCh)
	wg.Add(1)

	go func(invalidOrderCh <-chan invalidOrder) {
		for order := range invalidOrderCh {
			fmt.Printf("Invalid order received: %v, issue: %v", order.order, order.err)
		}
		wg.Done()
	}(invalidOrderCh)

	const workers = 3
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(reservedInventoryCh <-chan order) {
			for order := range reservedInventoryCh {
				fmt.Printf("Inventory reserved for: %v\n", order)
			}
			wg.Done()
		}(reservedInventoryCh)	
	}
	wg.Wait()
}
```

Above all the three workers have same exit strategy, as soon as we close the reservedInventoryChannel all three groroutines are going to terminate. So multiple consumers in go are really simple, we just need to spawn multiple consumers. But we lose ordering of the messages as these consumers might consume orders in any order. So it's fine if ordering doesn't matter.

### Multiple Producer & Single Consumer
For this we are going to add another worker called fillOrders.

```go
func fillOrders(in <-chan order) <-chan order {
	out := make(chan order)
	go func() {
		for o := range in {
			o.Status = filled
			out <- o
		}
		close(out)
	}()
	return out
}
```

We are going to spawn multiple producers in reservedInventory.

```go
func reserveInventory(in <-chan order) <-chan order {
	out := make(chan order)
	const workers = 3
	for i := 0; i < workers; i++ {
		go func() {
			for o := range in {
				o.Status = reserved
				out <- o
			}
			close(out)
		}()	
	}
	return out
}
```

So reserved inventory is multiple consumer of valida orders and multiple producer of reserved order which is consumed by single consumer i.e. fillOrder goroutine.

But the challenge is, each of these producer goroutines are going to send its messages and then its going to close the **out** channel. IF we run our program we can see a problem **we are trying to close already closed channel** or **we are trying to send on close channel** both of these cause panics. So we need to be little bit careful when adding multiple producers. We can't close the channel inside the Goroutine because we got three attempts to close the out channel and only one will be able to do it.

So how to fix it...So the answer is creation of another Goroutine. Add waitgroup and wait inside a Goroutine for all the workers to finish and then close the channel

```go
// messages coming in from valid order channel.
// output orders from receive only channel, as we don't want consumers to push messges
// into the channel
func reserveInventory(in <-chan order) <-chan order {
	out := make(chan order)
	var wg *sync.WaitGroup
	const workers = 3
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			for o := range in {
				o.Status = reserved
				out <- o
			}
			wg.Done()
		}()
	}

	go func ()  {
		wg.Wait()
		close(out)
	}()
	return out
}
```

### Multiple Producer & Multiple Consumer
We have got multiple producer set up in reservedInventory. We can add multiple workers in fillOrder.

```go
func fillOrders(in <-chan order) <-chan order {
	const workers = 3
	out := make(chan order)
	for i := 0; i < workers; i++ {
		go func() {
			for o := range in {
				o.Status = filled
				out <- o
			}
			close(out)
		}()	
	}
	return out
}
```

However, this will create a problem because we have got multiple producer/consumer and single consumer in fillOrder. So like we did earlier **we need to have supervisory Goroutine that waits for all workers to finish before closing the channel**. 

However, that's not even needed, we don't need to produce anything in fillOrder as it's the end of the workflow and since there is no consumer of fillOrders, we can remove the out channel. We can modify fillOrder as below:

```go
func fillOrders(in <-chan order) {
	const workers = 3
	for i := 0; i < workers; i++ {
		go func() {
			for o := range in {
				o.Status = filled
				fmt.Printf("Order has been completed: %v \n", o)
			}
		}()	
	}
}
```
main.go
```go
func main() {
	var wg sync.WaitGroup
	receivedOrderCh := receiveOrders()
	validOrderCh, invalidOrderCh := validateOrders(receivedOrderCh)
	reservedInventoryCh := reserveInventory(validOrderCh)
	fillOrders(reservedInventoryCh)
	wg.Add(1)

	go func(invalidOrderCh <-chan invalidOrder) {
		for order := range invalidOrderCh {
			fmt.Printf("Invalid order received: %v, issue: %v\n", order.order, order.err)
		}
		wg.Done()
	}(invalidOrderCh)
	wg.Wait()
}
```

However, we have a problem since we are not waiting on filledOrderCh, we might not see all the outputs. So to fix that we can pass the pointer to waitgroup and wait.

```go
func main() {
	var wg sync.WaitGroup
	receivedOrderCh := receiveOrders()
	validOrderCh, invalidOrderCh := validateOrders(receivedOrderCh)
	reservedInventoryCh := reserveInventory(validOrderCh)
	fillOrders(reservedInventoryCh, &wg)
	
	wg.Add(1)

	go func(invalidOrderCh <-chan invalidOrder) {
		for order := range invalidOrderCh {
			fmt.Printf("Invalid order received: %v, issue: %v\n", order.order, order.err)
		}
		wg.Done()
	}(invalidOrderCh)
	wg.Wait()
}

func fillOrders(in <-chan order, wg *sync.WaitGroup) {
	const workers = 3
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			for o := range in {
				o.Status = filled
				fmt.Printf("Order has been completed: %v \n", o)
			}
            wg.Done()
		}()
	}
}
```

Here is the complete program:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	receivedOrderCh := receiveOrders()
	validOrderCh, invalidOrderCh := validateOrders(receivedOrderCh)
	reservedInventoryCh := reserveInventory(validOrderCh)
	fillOrders(reservedInventoryCh, &wg)

	wg.Add(1)

	go func(invalidOrderCh <-chan invalidOrder) {
		for order := range invalidOrderCh {
			fmt.Printf("Invalid order received: %v, issue: %v\n", order.order, order.err)
		}
		wg.Done()
	}(invalidOrderCh)

	wg.Wait()
}

func fillOrders(in <-chan order, wg *sync.WaitGroup) {
	const workers = 3
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			for o := range in {
				o.Status = filled
				fmt.Printf("Order has been completed: %v \n", o)
			}
			wg.Done()
		}()
	}
}

func receiveOrders() <-chan order {
	out := make(chan order)
	go func() {
		for _, rawOrder := range rawOrders {
			var newOrder order
			err := json.Unmarshal([]byte(rawOrder), &newOrder)
			if err != nil {
				log.Println(err)
				continue
			}
			out <- newOrder
		}
		close(out)
	}()
	return out
}

// messages coming in from valid order channel.
// output orders from receive only channel, as we don't want consumers to push messges
// into the channel
func reserveInventory(in <-chan order) <-chan order {
	out := make(chan order)
	var wg sync.WaitGroup
	const workers = 3
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			for o := range in {
				o.Status = reserved
				out <- o
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func validateOrders(in <-chan order) (<-chan order, <-chan invalidOrder) {
	out := make(chan order)
	errch := make(chan invalidOrder, 1)
	go func() {
		for order := range in {
			if order.Quantity <= 0 {
				errch <- invalidOrder{
					order: order,
					err:   fmt.Errorf("quantity must be positive %f", order.Quantity),
				}
			} else {
				out <- order
			}
		}
		close(out)
		close(errch)
	}()
	return out, errch
}

var rawOrders = []string{
	`{"productCode": 1111, "quantity": -5, "status": 1}`,
	`{"productCode": 2222, "quantity": 42.3, "status": 1}`,
	`{"productCode": 3333, "quantity": 19, "status": 1}`,
	`{"productCode": 4444, "quantity": 8, "status": 1}`,
}
```

# Additional Tools
## The sync Package
sunc package has several features that are very useful, given specific situations when you are working concurrently in Go. However, there are two elements of the sync package that we are going to give special attention:

- Mutexes
- Once

### Mutexes
It stands for mutual exclusion lock and it works something like this. If we take the below example:

```go
var m sync.Mutex

func main() {
    go func() {
        m.Lock()
        defer m.Unlock()
        // do thing
    }
    go func() {
        m.Lock()
        defer m.Unlock()
        // do another thing
    }()
}
```

If scheduler hits the first Goroutine first then the first statement that it's going to see is the Lock method call. Now let's just imagine that before the scheduler is able to move on to the next line of code, we run into the Lock statement in other Goroutine. So this one is trying to acquire the lock as well, what the scheduler will do is **it will decide that one of those goroutines can continue and it will block the other one**.

Let's say first Goroutine was selected to run, it can go on to its execution code and do it's thing and then after it's done and Goroutine exits our deferred Unlock method executes. This allow the scheduler to allow the second Goroutine to do it's work and then it can Unlock the mutex at the end.

What this allows us to do is, this introduces a mechanism to synchronize or make Goroutines execute sequentially. SInce both of these Goroutines are contending for the lock, they are both trying to acquire the lock for the mutex at the same time, the scheduler is going to allow one to proceed. That means these Goroutines are no longer able to execute in parallel.

**We often use this in situations where we have shared memory that needs to be accessed by multiple Goroutines, but we don't want to have race conditions.**. We don't want to have multiple Goroutines working on the shared memory at the same time.

Let's look at one example to see it in action:
```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	s := []int{}
	var wg sync.WaitGroup
	// var m sync.Mutex

	const iterations = 1000
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			s = append(s, 1)
			wg.Done()
		}()

	}
	wg.Wait()
	fmt.Println(len(s))
}
```

If we run this we get random output less than 1000 and we have unpredictable condition logically that number should be 1000. So what's going on here?
The problem is we are accessing shared memory variable, s. We've got 1000 Goroutines, actually 1001 because the main function is accessing s as well. We've got 1001 goroutines that are all working with s at the same time.

So what happens occasionally is that the scheduler in the middle of the append operation is actually going to have another Goroutine with an append operation happen at the same time and only one of them is going to count because they're both going to overwrite s. So occasionally, we drop memory. Ocassionally we drop one of those operations, and we get this unpredictable behaviour.

So what's happening here is we've got shared memory that is being acted upon in parallel by Goroutines, and that leads to this kind of behaviour where now our program is unstable and unreliable, so how do we address this?
The one that we want to highlight right now is the use of a mutex. The mutex is going to allow us to take all of these goroutines that may run in parallel, depending on what the scheduler decides and we're going to force them to run sequentially.

```go
func main() {
	s := []int{}
	var wg sync.WaitGroup
	var m sync.Mutex

	const iterations = 1000
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			m.Lock()
			defer m.Unlock()
			s = append(s, 1)
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println(len(s))
}
```

Now we loose all the benifits of parallelism. So while mutexes are very, very good at protecting shared memory. There is a performance penalty that you're going to pay.

So in general, **we try and avoid shared memory wherever we can with Go because we don't want to force our Goroutines to run sequentially.**

### Once
It ensures that something happens one time. Why would you want to do that?
To explain it we are going to use a real life usecase. So we are going to pull sqlite package:

```sh
go get github.com/mattn/go-sqlite3
```

```go
var db *sql.DB
```

That db object acts kind of like the proxy of the database within our Go program, and we have to initialize that. So very often that initialization is done within a package when that package is first used. We could use init function to do that, but **there are some testablity concerns with init**.

So one of the patterns we often use in Go is we'll create a function and we'll have an internal convention. E.g. we got a function called Run, and when we see that in our organization, we know that the Run function should be called to initialize the package. Again, it serves the same purpose as init functions, but it's little bit more robust in testing situations.

```go
package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	Run()
}

var db *sql.DB

func Run() {
	log.Println("Opening the connection to database")
	var err error
	db, err = sql.Open("sqlite2", "./mydb.db")
	if err != nil {
		log.Fatal(err)
	}
}
```

What happens if someone calls Run again by mistake?
```go
func main() {
	Run()
    Run()
}
```

We see that "Opening the connection to database" happens twice and that's inefficient. Because now we are reinstantiating this database object which doesn't gain us anything and potentially causes us some problems if for example, we have some queries in flight, there might be some unpredictable behaviours. So ideally we want to make sure that this initialization happens only one time.

```go
package main

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	Run()
	Run()
}

var db *sql.DB
var o sync.Once

func Run() {
	o.Do(func() {
		log.Println("Opening the connection to database")
		var err error
		db, err = sql.Open("sqlite3", "./mydb.db")
		if err != nil {
			log.Fatal(err)
		}
	})
}
```

So once makes sure that the logic runs only once. So, this is commonly used in initialization logic where we want to make sure that, yes things do need to be initialized, but they should not be executed twice as they could cause some instability in our application or because that redundancy would be inefficient for some reason.

## Go's built-in Race Detector
So let's take below example:

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	s := []int{}
	var wg sync.WaitGroup
	// var m sync.Mutex

	const iterations = 1000
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			s = append(s, 1)
			wg.Done()
		}()

	}
	wg.Wait()
	fmt.Println(len(s))
}
```

If you recall we don't get length of 1000, we get some number less than that. And the reason for that is because we've got a data race condition in our application. One question might comeup is how do you find those?

In small examples it's pretty easy to spot it but may be not so easy in big applications. Fortunately Go has some tooling in place that's going to help us.

```sh
go run -race .
```

It impacts the performance because Go is constantly looking for thos rase conditions.

```
WARNING: DATA RACE
Write at 0x00c00009a0a8 by goroutine 61:
  main.main.func1()
      /Users/v.mishra/go/src/github.com/vivekprm/go-concurrency-pluralsight/main.go:17 +0x90

Previous write at 0x00c00009a0a8 by goroutine 57:
  main.main.func1()
      /Users/v.mishra/go/src/github.com/vivekprm/go-concurrency-pluralsight/main.go:17 +0x90

Goroutine 61 (running) created at:
  main.main()
      /Users/v.mishra/go/src/github.com/vivekprm/go-concurrency-pluralsight/main.go:16 +0x98

Goroutine 57 (running) created at:
  main.main()
      /Users/v.mishra/go/src/github.com/vivekprm/go-concurrency-pluralsight/main.go:16 +0x98
==================
674
Found 4 data race(s)
exit status 66
```

So Go found 4 data races with status 66, which tells there is data race. And all these dataraces happening at append.

**It only detects race conditions that occurs, so if you have a race condtion that occurs only periodically or only in certain set of circumstances the race detector will only find the race if it actually encounters it.**

## Context
When Go was first released, we had almost complete set of tools to handle concurrent development. But there was one thing that was determined to be missing relatively quickly, and that was addressed with the context package.

### The Purpose of Context
So let's look at the schematic of a possible Go program to see what the problem is that context were designed to solve.

We might have a function and that function might kickoff multiple Goroutines. So e.g. in this case func1() is kicking off 3 Goroutines func2, func3, & func4. Those Goroutines might kickoff their own Goroutines.

So now we have developed this tree of concurrency and because Goroutines are unrelated in anyway to the function that spawned them there's no way to communicate between them except using channels or shared memory.

However looking at this illustration, what happens if something occurs during the execution of func2 where it's no longer needed? So may be for example it completed it's execution ans we've got the answer that we were looking for or an error occured and we recognized that we could no longer proceed with this function. We need to do some remediation. What happens to Goroutine 5,6 & 7? 
Well before the context package they just kept executing because Go doesn't necessarily know that the results of their activity is not needed anymore and so we ended up wasting resources because we couldn't communicate the cancellation in func2 effecitvely to func5, func6 & func7.

Could you have done that using channels?
The answer is yes, kind of. Because channels while we can send messages from one Goroutine to another the challenge becomes how many cancellation messages should we send?
Now it would be reasonable to assume that goroutine2 knows about the three goroutines it spawned. But what happens if cancellation happens at func1? Now it knows about it's three Goroutines that it spawned, but it doesn't know about the Goroutines that were spawned as children to those first three Goroutines.

So **context** package was developed as a way to communicate calcellation between Goroutines. So when we use context package we endup with this model.

We can communicate using the context that the operation is no longer necessary and the context will communicate that cancellation over to the other goroutines.

So how does it work?
There are two basic forms
- Simple Cancellation
- Slightly Augmented One that allows us to setup a timer that will automatically cancel the context when that occurs.

https://pkg.go.dev/context@go1.23.3

If we look at context package it's very small, it has only five public members and couple of variables.

It has a Context type which is interface. So how do we create a Context?
We could create a type that implements this interface but that would be pretty inconcenient everytime. So instead what we do is we create a context by creating what's called **derived context**.

So we derive a new context from an existing context. What that allows us to do is just like we can have a chain of Goroutines that form a tree, we can have a chain of contexts that form a tree. So if one of the upstream context cancels, it communicates that cancellation down to all contexts that are derived from it. This allows us to add increasingly rigid constraints onto our context so that we can cacel out subtrees.

- ```func WithCancel(parent Context) (ctx Context, cancel CancelFunc)```
  - It takes one parameter, that's the parent context and then it returns two values, it returns the derived context and a cancel function. And a **cancel** function is simply a function that cancels the context.

Tick function returns a channel that receives a signal every duration.

```go
func main() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for range time.Tick(500 * time.Millisecond) {
			fmt.Println("tick")
		}
		wg.Done()
	}()
	wg.Wait()
}
```

This is one way of setting intervals in your Go program. Now we want to use a context to communicate when I want this Goroutine to stop, to be canceled.

The **Background Context** represents the context that exits throughout the execution of our program. In other words it's a context that never cancelled because as soon as it would cancel, our program stops executing. So **the background context is only cancelled when the program stops**.

**TODO** is the same thing but **TODO** represents a placeholder. So we are going to use TODO in a place where we should have a parent context coming to us, but we don't have one yet. So it's intended to be sort of a documentation mechanism to communicate to us in the future and to the rest of our team that a parent context with constraints should already be provided to us.

We always pass the ctx into Goroutines we don't tend to use them as Closures. When you have context aware function, we always have context as first parameter. That's the best practice and the intention for that is to make it immediately obvious that this function is working with context.

```go
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func(ctx context.Context) {
		for range time.Tick(500 * time.Millisecond) {
			fmt.Println("tick")
		}
		wg.Done()
	}(ctx)

	time.Sleep(2 * time.Second)
	cancel()

	wg.Wait()
}
```

If we run the program as it is, it doesn't change the behaviour at all and works as it was working without context and it didn't cancel anything and the reason is because we actually have to check if the context is cancelled. So cancelling the context doesn't necessarliy mean that the Goroutines should immediately terminate what they're doing. They might be in the middle of some critical process that they have to finish or they might need to cleanup resources in order to have a nice clean exit.

So the context object will communicate cancellation, but we do have to inspect it.

```go
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func(ctx context.Context) {
        defer wg.Done()
		for range time.Tick(500 * time.Millisecond) {
			// Context cancelled
			if ctx.Err() != nil {
				log.Println(ctx.Err())
				return
			}
			fmt.Println("tick")
		}
	}(ctx)
	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()
}
```

So this is the basic way we can communicate cancellation. If we derive anymore context from the one that we created, those will be cancelled when this one is cancelled.

- ```func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)```
A very common usecase that we have with contexts is we want establish some sort of a deadline for an activity. So may be we are querying a database but we don't want to lock up the database with unending query if that SQL stement has been malformed. So we might want to send a query and set a timeout for maybe 30 seconds.