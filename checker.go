package checker

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	content, _ := ioutil.ReadFile("url_list.txt")
	urls := strings.Split(string(content), "\n")

	const workers = 25

	wg := new(sync.WaitGroup)
	in := make(chan string, 2*workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range in {
				URLTest(url)
			}
		}()
	}

	for _, url := range urls {
		if url != "" {
			in <- url
		}
	}
	close(in)
	wg.Wait()
}

func URLTest(url string) (time.Duration, int) {
	req, err := http.NewRequest("GET", url, nil)

	// Starting the benchmark
	timeStart := time.Now()

	resp, err := http.DefaultTransport.RoundTrip(req)

	if err != nil {
		log.Printf("Error fetching: %v", err)
		return 0, 500
	}
	defer resp.Body.Close()

	// How long did it take
	duration := time.Since(timeStart)

	fmt.Println(duration, url, " Status code: ", resp.StatusCode)

	return duration, resp.StatusCode
}
