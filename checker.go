package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

type SmtpTemplateData struct {
	From    string
	To      string
	Subject string
	Body    string
}

type BrokenPage struct {
	Url        string
	StatusCode int
}

func main() {
	brokenPages := []BrokenPage{}

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
				code, _, _, _ := URLTest(url)

				if code != 200 {
					brokenPages = append(brokenPages, BrokenPage{url, code})
				}
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

	if len(brokenPages) > 0 {
		fmt.Printf("%d broken pages detected\n\n", len(brokenPages))
		message := "Success"

		for _, row := range brokenPages {
			if row.StatusCode == 500 {
				message = "Alert"
			} else if row.StatusCode == 404 {
				message = "Fatal error"
			} else {
				message = "Warning"
			}

			sendEmail(row.Url, message)
		}
	}
}

func URLTest(url string) (int, time.Duration, string, string) {
	req, err := http.NewRequest("GET", url, nil)

	// Starting the benchmark
	timeStart := time.Now()

	resp, err := http.DefaultTransport.RoundTrip(req)

	if err != nil {
		log.Printf("Error fetching: %v", err)

		return 404, 0, "", ""
		// panic()
	}
	defer resp.Body.Close()

	content, _ := ioutil.ReadAll(resp.Body)
	contentType := resp.Header.Get("Content-Type")

	// How long did it take
	duration := time.Since(timeStart)

	fmt.Println(duration, url, " Status code: ", resp.StatusCode)

	return resp.StatusCode, duration, contentType, string(content)
}

func sendEmail(website string, messageType string) {
	var err error

	smtpSrv := "smtp.gmail.com"
	to := []string{"twl@phase-ii.com"}
	msg := []byte("To: " + strings.Join(to, ", ") + "\r\n" +
		"Subject: PING " + messageType + " for " + website + " !!\r\n" +
		"\r\n" +
		"Hi Phase II team member,\n\n" +
		"This is a notification sent by Ping®.\n\n" +

		"Incident (" + messageType + ") for `" + website + "`, has been assigned to you.\n\n" +
		"You will be notified when the page goes live back again.\n\n" +

		"Best regards,\n" +
		"The Phase II Team.\n" +
		"Tomek Wlodarczyk\r\n")

	auth := smtp.PlainAuth("",
		"fromUsername",
		"fromPassword",
		smtpSrv,
	)

	err = smtp.SendMail(
		smtpSrv+":587",
		auth,
		"from",
		to,
		msg,
	)

	if err != nil {
		log.Print("ERROR: attempting to send a mail ", err)
	}

	fmt.Println("Notification [" + messageType + "] sent to " + strings.Join(to, ", "))
}
