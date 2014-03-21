// gofaster project main.go
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"
)

type Requester struct {
	stopChan chan bool

	totalCount   int
	successCount int
	failCount    int

	// Average response time counted in microseconds
	averageResponse int

	runtime time.Duration

	client *http.Client
}

func (r *Requester) start(requestDone chan bool, rg RequestGenerator) {
	r.client = &http.Client{}
	startTime := time.Now()
	for {
		select {
		case <-r.stopChan:
			r.runtime = time.Now().Sub(startTime)
			requestDone <- true
			return

		default:
			req, err := rg.GenerateRequest()
			r.totalCount++

			if err != nil {
				r.failCount++
				fmt.Println(err)
				return
			}

			reqStart := time.Now()

			resp, err := r.client.Do(req)
			if err != nil {
				r.failCount++
				continue
			}
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()

			reqEnd := time.Now()

			micS := reqEnd.Sub(reqStart) / time.Microsecond

			allTimes := int64(r.averageResponse) * int64(r.successCount)
			r.averageResponse = int((allTimes + int64(micS)) / int64(r.successCount+1))

			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				r.successCount++
			} else {
				r.failCount++
			}
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
	TotalRuntime time.Duration

	AverageResponse time.Duration
}

func (rs *RequestSet) run(rg RequestGenerator) {
	rs.requesters = make([]Requester, rs.ParallelRequests)
	rs.requestDone = make(chan bool, rs.ParallelRequests)

	for i := 0; i < rs.ParallelRequests; i++ {
		req := &rs.requesters[i]
		req.stopChan = make(chan bool, 1)
		go req.start(rs.requestDone, rg)
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

	avgResp := int64(0)
	for _, req := range rs.requesters {
		rs.TotalCount += req.totalCount
		rs.FailCount += req.failCount
		rs.SuccessCount += req.successCount
		rs.TotalRuntime += req.runtime
		avgResp += int64(req.averageResponse)
	}
	rs.AverageResponse = time.Duration(avgResp/int64(rs.ParallelRequests)) * time.Microsecond
}

func (rs *RequestSet) SuccessfulRequestsPerSecond() float32 {
	return float32(rs.SuccessCount) / float32(rs.TotalRuntime/time.Second) * float32(rs.ParallelRequests)
}

func clearLine() {
	fmt.Print("                                                  \r")
}

func printStats(rs *RequestSet) {
	clearLine()

	fmt.Printf(
		"Threads %d\tSuccessfull %d (%.2f/s)\tFailed %d (%.2f%%)\tAvg time %dms\n",
		rs.ParallelRequests,
		rs.SuccessCount, rs.SuccessfulRequestsPerSecond(),
		rs.FailCount, float32(rs.FailCount)/float32(rs.TotalCount)*100,
		int(rs.AverageResponse/time.Millisecond),
	)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var lastRs *RequestSet = nil
	generator := StaticRequestGenerator{"http://localhost/sleep.php"}

	for i := 1; ; i *= 2 {
		rs := RequestSet{
			ParallelRequests: i,
			TimeToRun:        10,
		}
		rs.run(&generator)
		printStats(&rs)

		if lastRs != nil && lastRs.SuccessCount > rs.SuccessCount {
			break
		}

		lastRs = &rs
	}
}
