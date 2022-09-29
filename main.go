package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func doWork(ch chan<- int, i int, wg *sync.WaitGroup) {
	time.Sleep(time.Duration(rand.Intn(1000)) * 10 * time.Millisecond)
	ch <- i
	wg.Done()
}

// runWorkers wg param must be passed by reference to prevent copying sync.WaitGroup struct. If we allow this, wg.Done()
// will decrement not the right counter and execution of main app will go forever.
func runWorkers(amountOfWork int, chDoWork chan<- int, chDoAnotherWork chan<- int, wg *sync.WaitGroup) {
	wgWorkDone := sync.WaitGroup{}
	wgAntherWorkDone := sync.WaitGroup{}

	for i := 0; i < amountOfWork; i++ {
		wgWorkDone.Add(1)
		go doWork(chDoWork, i, &wgWorkDone)

		wgAntherWorkDone.Add(1)
		go doWork(chDoAnotherWork, i, &wgAntherWorkDone)
	}

	wg.Done()
}

func runWorkProcessing(ctx context.Context, chDoWork <-chan int, chDoAnotherWork <-chan int, wg *sync.WaitGroup) {
	for {
		select {
		case i := <-chDoWork:
			fmt.Printf("Doing some work #%d...\n", i)
		case i := <-chDoAnotherWork:
			fmt.Printf("Doing some another work #%d...\n", i)
		case <-ctx.Done(): // here, after call of cancelFunc() we will get signal from context and end endless execution loop
			fmt.Println("Done.")
			wg.Done() // After that the last wgGlobal.Wait() instruction
			// will unfreeze main goroutine and correctly ends execution of application
			return
		default:
			fmt.Println("Idling...")
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// getContext It is not the usual practice to use function to init context.
//In real app just init it in main if no further instructions were given
func getContext() (context.Context, context.CancelFunc) {
	ctxRoot := context.Background()
	//ctx, cancelFunc := context.WithTimeout(ctxRoot, 1*time.Second)

	//ctx, cancelFunc := context.WithDeadline(ctxRoot, time.Date(
	//	2022, 9, 27, 19, 48, 0, 0,
	//	time.UTC),
	//)

	//ctx := context.WithValue(ctxRoot, "key", 123)

	return context.WithCancel(ctxRoot)
}

// gracefulShutdown Synthetic stop mechanism. Halts all execution after a given timeout.
// In this case you can use context.Timeout instead
func gracefulShutdown(duration time.Duration, cancelFunc context.CancelFunc) {
	defer cancelFunc()
	// after calling cancelFunc() we will signal that execution in given context is done, and thus we can shut off app
	time.Sleep(duration)
	fmt.Printf("Gracefully shutdown after %f seconds\n", duration.Seconds())
}

func main() {
	ctx, cancelFunc := getContext()
	go gracefulShutdown(10*time.Second, cancelFunc) // run async our app's killer function

	wgGlobal := sync.WaitGroup{}

	chQueueSize := 2 // chQueueSize is lesser than amountOfWork for demonstrate buffered channels working correctly
	chDoWork := make(chan int, chQueueSize)
	chDoAnotherWork := make(chan int, chQueueSize)

	wgGlobal.Add(1)
	go runWorkers(10, chDoWork, chDoAnotherWork, &wgGlobal)

	wgGlobal.Add(1)
	go runWorkProcessing(ctx, chDoWork, chDoAnotherWork, &wgGlobal)

	fmt.Println("I'm still alive! And able to run another services too!")
	wgGlobal.Wait()
}
