// gofaster project main.go
package main

import (
	"fmt"
	"net/http"
	"time"
)

type Requester struct {
	stopChan     chan bool
	totalCount   int
	successCount int
	failCount    int

	client *http.Client
}

func (r *Requester) start(requestDone chan bool) {
	r.client = &http.Client{}
	for {
		select {
		case <-r.stopChan:
			requestDone <- true
			return

		default:
			req, err := http.NewRequest("GET", "http://localhost/sleep.php", nil)
			r.totalCount++

			if err != nil {
				r.failCount++
				fmt.Println(err)
				return
			}

			resp, err := r.client.Do(req)
			if err != nil {
				r.failCount++
				continue
			}

			r.successCount++
			resp.Body.Close()
		}
	}
}

type RequestSet struct {
	ParallelRequests int

	// Time to run in seconds
	TimeToRun int

	requestDone     chan bool
	requesters      []Requester
	runningRequests int

	TotalCount   int
	SuccessCount int
	FailCount    int
}

func (rs *RequestSet) run() {
	rs.requesters = make([]Requester, rs.ParallelRequests)
	rs.requestDone = make(chan bool, rs.ParallelRequests)

	for i := 0; i < rs.ParallelRequests; i++ {
		req := &rs.requesters[i]
		req.stopChan = make(chan bool, 1)
		go req.start(rs.requestDone)
		rs.runningRequests++
	}

	clearLine()
	fmt.Printf("%d threads running...\r", rs.ParallelRequests)

	time.Sleep(time.Duration(rs.TimeToRun) * time.Second)

	for i := 0; i < rs.ParallelRequests; i++ {
		req := &rs.requesters[i]
		clearLine()
		fmt.Printf("Stopping %d threads...\r", rs.ParallelRequests-i)
		req.stopChan <- true
	}

	for rs.runningRequests > 0 {
		clearLine()
		fmt.Printf("Waiting for %d threads to terminate...\r", rs.runningRequests)
		<-rs.requestDone
		rs.runningRequests--
	}

	for _, req := range rs.requesters {
		rs.TotalCount += req.totalCount
		rs.FailCount += req.failCount
		rs.SuccessCount += req.successCount
	}
}

func clearLine() {
	fmt.Print("                                                  \r")
}

func printStats(rs *RequestSet) {
	clearLine()

	fmt.Printf(
		"Threads %d\tSuccessfull %d (%.2f/s)\tFailed %d (%.2f%%)\tAvg time %dms\n",
		rs.ParallelRequests,
		rs.SuccessCount, float32(rs.SuccessCount)/float32(rs.TimeToRun),
		rs.FailCount, float32(rs.FailCount)/float32(rs.TotalCount)*100,
		0,
	)
}

func main() {
	var lastRs *RequestSet = nil
	for i := 8; ; i *= 2 {
		rs := RequestSet{
			ParallelRequests: i,
			TimeToRun:        5,
		}
		rs.run()
		printStats(&rs)

		if lastRs != nil && lastRs.SuccessCount > rs.SuccessCount {
			break
		}

		lastRs = &rs
	}
}
