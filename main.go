package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

func parralell(jobs int, fn func(j int)) {
	for job := 1; job <= jobs; job++ {
		log.Println("Starting job", job)
		go fn(job)
	}
}

func doStuff() error {
	rd := time.Duration(math.Mod(float64(rand.Int()), 5))
	time.Sleep(rd * time.Second)
	return nil
}

func cleanup(job int, msgs chan string) {
	doStuff()
	msgs <- fmt.Sprintf("Done cleaning up.. %d", job)
}

func main() {
	var wg sync.WaitGroup
	var gg sync.WaitGroup
	jobs := 100
	wg.Add(jobs)
	gg.Add(jobs)

	// pass messages between jobs
	msgs := make(chan string, jobs*2) // hacky, requires some way of estimating the buffer
	errCh := make(chan error)
	defer close(errCh)

	// signals
	done := make(chan bool)
	exit := make(chan bool, 1)
	defer close(exit)

	// flag for close on done channel
	doneClosed := false

	parralell(jobs, func(job int) {
		defer gg.Done()
		defer cleanup(job, msgs)
		defer wg.Done()

		select {
		case errCh <- doStuff():
			msgs <- fmt.Sprintf("Done with job %d", job)
		case <-done: // Gracefully exit incomplete job
			msgs <- fmt.Sprintf("Exiting job %d", job)
		}
	})

	// handles the case when all jobs finish before timeout
	go func() {
		log.Println("Blocking until all exits")
		wg.Wait()
		log.Println("All jobs exited, closing done channel")
		if !doneClosed { // close only if not already closed
			close(done)
			doneClosed = true
		}
	}()

	// handles the case when jobs do not finish before timeout
	go func() {
		<-time.After(2 * time.Second)
		if !doneClosed { // close only if not already closed
			log.Println("Timed out, exiting everything")
			close(done)
			doneClosed = true
		}
	}()

	// logger
	go func() {
		for {
			select {
			case msg := <-msgs:
				log.Println(msg)
			case err := <-errCh:
				if err != nil {
					log.Println(err)
				}
			case <-exit:
				return
			}
		}
	}()

	gg.Wait() // Everything aside from the logger should have gracefully exited by this point

	exit <- true // signal to logger to exit
	log.Println("Cleanup complete, closing msgs channel")
	close(msgs)
	log.Println("Draining messages")
	for msg := range msgs {
		log.Println(msg)
	}
	log.Println("Finished draining messages")
}
