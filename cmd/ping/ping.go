package main

/*
https://github.com/golang/tour/blob/master/solutions/webcrawler.go -> Real example of a web crawler script

@TODO: in theory if interval set to 1, it should check every single minute. But practically it's every 2 minutes
		because the seconds that system needs to check the ping break everything. The solution would be to keep
		the minutes only without the seconds in mongo (or ignoring the seconds when pinging)
@TODO: OFF/ON flag is needed to temporary disable an endpoint from pinging
*/

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/tomekwlod/ping"
	"github.com/tomekwlod/ping/db"
)

type fetchResult struct {
	URL         string
	Code        int
	Duration    time.Duration
	ContentType string
	Content     string
}

func mgoHost() (host string) {
	// Database host from the environment variables
	host = os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost:27017"
	}

	return
}

var (
	l *log.Logger
)

type service struct {
	session *mgo.Session
}

// functions for the service struct
func (s *service) getPageRepo() ping.IPageRepository {
	return &ping.PageRepository{Session: s.session.Clone()}
}
func (s *service) getPageEntryRepo() ping.IPageEntryRepository {
	return &ping.PageEntryRepository{Session: s.session.Clone()}
}

func main() {
	// definig the logger & a log file
	file, err := os.OpenFile("log/ping.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		l.Fatalln("Failed to open log file", err)
	}
	multi := io.MultiWriter(file, os.Stdout)
	l = log.New(multi, "", log.Ldate|log.Ltime|log.Lshortfile)

	// definging the mongodb session
	mgoSession, err := db.CreateSession(mgoHost())
	defer mgoSession.Close()
	if err != nil {
		l.Panic("Cannot connect to Mongodb: ", err)
	}

	// combine the datastore session and the logger into one struct
	s := &service{
		session: mgoSession}

	// do i have to open two sessions here?
	pageRepo := s.getPageRepo()           // clone the session
	defer pageRepo.Close()                // close the session defer
	pageEntryRepo := s.getPageEntryRepo() // clone the session
	defer pageEntryRepo.Close()           // close the session defer

	pages, err := pageRepo.PagesForPing()
	if err != nil {
		l.Panic(err)
	}

	if len(pages) == 0 {
		l.Println("No queued pages found")

		return
	}

	// When we know the number of goroutines to use we might count them to know when to finish. But then the waitgroup
	// is superfluous, confusing and overcomplicated. WaitGroups are more useful for doing different tasks in parallel, or when
	// we don't know how many goroutines we actually need (eg. recursive going through the directories)
	// Because of the above, below there are only goroutines and channels are used
	//
	// https://nathanleclaire.com/blog/2014/02/21/how-to-wait-for-all-goroutines-to-finish-executing-before-continuing-part-two-fixing-my-ooops/
	// https://www.reddit.com/r/golang/comments/1y3spq/how_to_wait_for_all_goroutines_to_finish/

	type response struct {
		page   *ping.Page
		result fetchResult
		err    error
	}

	// we create a channel here in a type of the 'response'
	// we need a channel here because we want a response from a goroutine back in a main func body
	// channels explained: https://programming.guide/go/channels-explained.html
	ch := make(chan response)

	for _, page := range pages {
		// we start a goroutine which expects a string parameter
		go func(p *ping.Page) {
			res, err := urlTest(p.Url)

			// using a channel we send the response outside of the goroutine
			ch <- response{p, res, err}
		}(page)
	}

	l.Println("Results:")

	results := []response{}

	// we now loop through the pages and collect the responses from the urls
	for range pages {
		// to be honest we don't need the channels here at all. The whole logic can be moved inside
		// the goroutines which would be even better, but for the learning purposes the channels are present here
		r := <-ch

		if r.err != nil {
			l.Printf("Error fetching %v: %v\n", r.result, r.err)
			continue
		}

		// we might collect the results or do the rest of the things just here
		results = append(results, r)

		l.Printf("\tCODE:%d\t%s\t%s\n", r.result.Code, r.result.Duration, r.result.URL)
	}

	if len(results) > 0 {
		for _, row := range results {

			// sending an email only if the last status code is 200 to avoid spaming
			if (row.page.LastStatus == 200 && row.result.Code != 200) || (row.page.LastStatus != 200 && row.result.Code == 200) {
				sendEmail(row.page.Url, row.result.Code)
			}

			content := ""
			if row.result.Code != 200 {
				content = row.result.Content
			}

			pageEntry := &ping.PageEntry{Code: row.result.Code, Load: row.result.Duration.Seconds(), Page: row.page.Id}
			pageEntry.SetInsertDefaults(time.Now())

			err := pageEntryRepo.Create(pageEntry)
			if err != nil {
				l.Panicln(err)
			}

			page := row.page
			page.LastStatus = row.result.Code
			page.Modified = time.Now()
			page.NextPing = time.Now().Add(time.Hour*time.Duration(0) + time.Minute*time.Duration(page.Interval) + time.Second*time.Duration(0))
			if content != "" {
				// update content only when error appears
				page.Content = content
			}

			err = pageRepo.Upsert(page)
			if err != nil {
				l.Panic(err)
			}
		}
	}
}

func urlTest(url string) (fetchResult, error) {
	if !strings.Contains(url, "http://") {
		url = "http://" + url
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fetchResult{}, err
	}

	// Starting the benchmark
	timeStart := time.Now()

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return fetchResult{}, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fetchResult{}, err
	}

	contentType := resp.Header.Get("Content-Type")

	// How long did it take
	duration := time.Since(timeStart)

	return fetchResult{url, resp.StatusCode, duration, contentType, string(content)}, nil
}

func sendEmail(url string, statusCode int) {
	// this should go to the main function to load it only once
	cnf, err := ping.LoadConfig()
	if err != nil {
		l.Fatalln(err)
	}

	if cnf.SMTP_Email == "" || cnf.SMTP_Server == "" || cnf.SMTP_Port == "" || len(cnf.SMTP_Emails) == 0 {
		l.Println("SMTP credentials not set. Skipping email notification")
		return
	}

	body := ""
	subject := ""
	if statusCode != 200 {
		message := "Warning"
		if statusCode == 500 {
			message = "Alert"
		} else if statusCode == 404 {
			message = "Fatal Error"
		}
		subject = "Incident OPEN (" + message + ") for " + url

		body = "Hi there,\n\n" +
			"This is a notification sent by Ping®.\n\n" +

			"Incident (" + message + ") for " + url + ", has been assigned to you.\n\n" +
			"You will be notified when the page goes live back again.\n\n" +

			"Best regards,\n" +
			"Ping®\r\n"
	} else {
		subject = "Incident CLOSED for " + url
		body = "Hi there,\n\n" +
			"This is a notification sent by Ping®.\n\n" +

			"Incident CLOSED for " + url + "\n\n" +

			"Best regards,\n" +
			"Ping®\r\n"
	}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = cnf.SMTP_Email
	headers["To"] = strings.Join(cnf.SMTP_Emails, ",")
	headers["Subject"] = subject

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server
	servername := cnf.SMTP_Server + ":" + cnf.SMTP_Port
	l.Println(servername)
	// host, _, _ := net.SplitHostPort(servername)

	c, err := smtp.Dial(servername)
	if err != nil {
		l.Panic(err)
	}

	// To && From
	if err = c.Mail(cnf.SMTP_Email); err != nil {
		l.Panic(err)
	}

	for _, email := range cnf.SMTP_Emails {
		if err = c.Rcpt(email); err != nil {
			l.Panic(err)
		}
	}

	// Data
	w, err := c.Data()
	if err != nil {
		l.Panic(err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		l.Panic(err)
	}

	err = w.Close()
	if err != nil {
		l.Panic(err)
	}

	c.Quit()

	fmt.Println("Notification sent to " + strings.Join(cnf.SMTP_Emails, ", "))
}
