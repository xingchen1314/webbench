package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Request struct {
	Text  string `json:"text"`
	Type  int    `json:"type"`
	Cache int    `json:"cache"`
}

var (
	ParallelNum = flag.Int("c", 1, " the number of parallel requests")
	ReqNum      = flag.Int("n", 1, "end time")
	Address     = flag.String("h", "http://127.0.0.1:7004/api", "host")
	Body        = flag.String("d", "", "request body")
)

func main() {
	flag.Parse()
	if *ParallelNum == 0 {
		BenchReqeust(1, *ReqNum)
		return
	}

	BenchReqeust(*ParallelNum, *ReqNum)
}

func trace(msg string) func() {
	start := time.Now()
	fmt.Printf("enter time %s\n", msg)
	return func() {
		fmt.Printf("exit %s (%s) \n", msg, time.Since(start))
	}
}

var TotalResponseCount int64

func BenchReqeust(ParallelNum int, ReqNum int) {
	defer trace("BenchReqeust")()
	if ParallelNum == 1 {
		for i := 0; i < ReqNum; i++ {

			fmt.Println("请求体:", string(*Body))

			ret, err := HttpPost([]byte(*Body), *Address)
			if err != nil {
				fmt.Println("error:", err.Error())
			}
			atomic.AddInt64(&TotalResponseCount, 1)
			fmt.Println("结果：", string(ret), "Count:", atomic.LoadInt64(&TotalResponseCount))
		}
		return
	}

	group := sync.WaitGroup{}
	if ParallelNum == 0 && ReqNum == 0 {
		panic("param error")
	}

	for i := 0; i < ParallelNum; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			for j := 0; j < ReqNum; j++ {

				fmt.Println("请求体:", string(*Body))

				ret, err := HttpPost([]byte(*Body), *Address)
				if err != nil {
					fmt.Println("error:", err.Error())
					continue
				}
				atomic.AddInt64(&TotalResponseCount, 1)
				fmt.Println("结果：", string(ret), "Count:", atomic.LoadInt64(&TotalResponseCount))
			}
		}(i)
	}
	group.Wait()
}

var HttpClient *http.Client

func init() {
	trans := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   100,
	}

	HttpClient = &http.Client{
		Transport: trans,
		Timeout:   30 * time.Second,
	}
}

func HttpPost(body []byte, url string) ([]byte, error) {

	reader := bytes.NewReader(body)

	var respBytes []byte
	request, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return respBytes, err
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := HttpClient.Do(request)
	if err != nil {
		return respBytes, err
	}
	var resErr error
	respBytes, resErr = ioutil.ReadAll(resp.Body)
	if resErr != nil {
		return respBytes, resErr
	}
	return respBytes, nil
}
