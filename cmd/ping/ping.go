package main

/*
https://github.com/golang/tour/blob/master/solutions/webcrawler.go -> an example of a web crawler script

@TODO: in theory if interval set to 1, it should check every single minute. But practically it's every 2 minutes
		because the seconds that system needs to check the ping break everything. The solution would be to keep
		the minutes only without the seconds in mongo (or ignoring the seconds when pinging)
@TODO: OFF/ON flag is needed to temporary disable an endpoint from pinging
@TODO: Interval value should be taken only when the last status is 200. Otherwise always take the url to test
@TODO: Emails should be sent only if the site is down for more than 10min (optionally)
@TODO: There may be a good idea to assign emails to urls (so project1 notifications will be sent only to email1 & email2)
*/

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
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

type response struct {
	page   *ping.Page
	result fetchResult
	err    error
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
	// definig the logger & the log file
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

	// we create a channel here in a type of the 'response'
	// we need a channel here because we want a response from a goroutine back in a main func body
	// channels explained: https://programming.guide/go/channels-explained.html
	ch := make(chan response)

	for _, page := range pages {
		// we start a goroutine which expects a string parameter
		go func(p *ping.Page) {
			res, err := urlTest(p.Url)

			// we could do the rest of the work here, but for the learning purposes i used the channels
			// to do it outside of the goroutine

			// using a channel we send the response outside of the goroutine
			ch <- response{p, res, err}
		}(page)
	}

	// results := []response{}

	// we now loop through the pages and collect the responses from the urls
	for range pages {
		// to be honest we don't need the channels here at all. The whole logic can be moved inside
		// the goroutines which would be even better, but for the learning purposes the channels are present here
		r := <-ch

		// if r.err != nil {
		// 	// maybe just comment out this statement? we have to also flag the broken pages, but with this
		// 	// statement we are not doing it at all
		// 	l.Printf("Error fetching %v: %v\n", r.result, r.err)
		// 	continue
		// }

		l.Printf("\tCODE:%d\t%s\t%s\n", r.result.Code, r.result.Duration, r.result.URL)

		// sending an email only if the status from the result doesn't match the last page status
		if pageUnstable(r) == true {
			sendEmail(r.page, r.result.Code)
		}

		updatePage(r, pageRepo, pageEntryRepo)
	}
}

func pageUnstable(r response) bool {
	if (r.page.LastStatus == 200 && r.result.Code != 200) || (r.page.LastStatus != 200 && r.result.Code == 200) {
		return true
	}

	return false
}

func updatePage(r response, pageRepo ping.IPageRepository, pageEntryRepo ping.IPageEntryRepository) {
	content := ""
	if r.result.Code != 200 {
		content = r.result.Content
	}

	pageEntry := &ping.PageEntry{Code: r.result.Code, Load: r.result.Duration.Seconds(), Page: r.page.Id}
	pageEntry.SetInsertDefaults(time.Now())

	err := pageEntryRepo.Create(pageEntry)
	if err != nil {
		l.Panicln(err)
	}

	page := r.page
	page.LastStatus = r.result.Code
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

func urlTest(url string) (fetchResult, error) {
	// if !strings.Contains(url, "http://") {
	// 	url = "http://" + url
	// }

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

func sendEmail(page *ping.Page, statusCode int) {
	url := page.Url

	// this should go to the main function to load it only once
	cnf, err := ping.LoadConfig()
	if err != nil {
		l.Fatalln(err)
	}

	if cnf.SMTP_Email == "" || cnf.SMTP_Server == "" || cnf.SMTP_Port == "" || len(cnf.SMTP_Emails) == 0 {
		l.Println("SMTP credentials not set. Skiping email notification")
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

		subject = "Incident [" + strconv.Itoa(statusCode) + "] for " + url
		body = `Incident OPENED for ` + page.Name + `!
Find the details below and instructions to fix the issue

Url: ` + url + `
Message: ` + message + `
Status code: ` + strconv.Itoa(statusCode) + `
Description: ` + page.Description + `

You can see all the endpoints here: ` + os.Getenv("GUI_ADDR") + `

You will be notified when the page goes live back again.

Ping®
`
	} else {
		subject = "Incident CLOSED for " + url
		body = `Incident CLOSED for ` + page.Name + `

Url: ` + url + `
Downtime: ...

You can see all the endpoints here: ` + os.Getenv("GUI_ADDR") + `

Ping®
`
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
