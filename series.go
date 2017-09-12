package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const tickerTime = 250
const factorialEndpoint = "http://localhost:12345/"

func generateNumber(ch chan<- int, close <-chan bool, t *time.Ticker, waitgroup *sync.WaitGroup) {
	defer waitgroup.Done()
	number := 0
	// Iterate and produce some numbers
	for {
		select {
		case <-t.C:
			ch <- number
			number++
		case <-close:
			return
		}
	}
}

type result struct {
	N int
	A big.Int
}

func getFactorialWorker(in <-chan int, out chan<- result, client *http.Client, waitgroup *sync.WaitGroup) {
	defer waitgroup.Done()
	for v := range in {
		r := getFactorialResult(v, client)
		if r == nil {
			continue // ignoring errors for the moment
		}
		out <- result{v, *r} // just a single number
	}
}

func getFactorialResult(n int, client *http.Client) *big.Int {
	// Creat a new request
	req, _ := http.NewRequest("GET", factorialEndpoint, nil)
	req.Header.Add("Content-type", "text/plain")
	q := req.URL.Query()
	q.Add("n", strconv.Itoa(n))
	val := q.Encode()
	req.URL.RawQuery = val

	// Get the factorial number
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error getting factorial result from server.", err)
		return nil // couild handle these errors better
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)

	r, ok := new(big.Int).SetString(string(contents), 10)
	if !ok {
		fmt.Println("Failed to convert number from server response. ", err)
	}
	return r
}

func calculateFirstTenAysnc(client *http.Client) chan result {
	ch := make(chan result, 10)
	for i := 0; i <= 9; i++ {
		go func(n int) {
			r := getFactorialResult(n, client)
			ch <- result{n, *r}
		}(i)
	}
	return ch
}

func partOne() {
	fmt.Println("Starting factorial test part #1")
	ticker := time.NewTicker(tickerTime)
	tickerManager := make(chan bool)
	numberChannel := make(chan int)
	responseChannel := make(chan result)

	waitgroup := &sync.WaitGroup{}
	waitgroup.Add(2) // we only have two routines at the moment

	// Spin off a ticker
	go generateNumber(numberChannel, tickerManager, ticker, waitgroup)
	// Could reuse the client across channels
	go getFactorialWorker(numberChannel, responseChannel, &http.Client{}, waitgroup)

	// Register the SIGKILL hook
	killChannel := make(chan os.Signal, 2)
	signal.Notify(killChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		_ = <-killChannel // blocking; when we get it we do stuff
		fmt.Println("Cleaning up and shutting down.")
		// Clean up the channels and the ticker
		ticker.Stop()
		tickerManager <- false
		// Once the goroutines finish we can safely close the channels
		close(numberChannel)
		close(tickerManager)
		close(responseChannel)
		waitgroup.Wait()
	}()

	for result := range responseChannel {
		fmt.Println("N: ", result.N, ", A: ", result.A.String())
	}

	fmt.Println("See you later.")
}

func partTwo() {
	ch := calculateFirstTenAysnc(&http.Client{})

	// Register the SIGKILL hook
	killChannel := make(chan os.Signal, 2)
	signal.Notify(killChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		_ = <-killChannel // blocking; when we get it we do stuff
		fmt.Println("Cleaning up and shutting down.")
		close(ch)
	}()

	for result := range ch {
		fmt.Println("N: ", result.N, ", A: ", result.A.String())
	}
}

func main() {
	//partOne()
	partTwo()

}
