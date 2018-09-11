package main

/*
@TODO: in theory if interval set to 1, it should check every single minute. But practically it's every 2 minutes
		because the seconds that system needs to check the ping break everything. The solution would be to keep
		the minutes only without the seconds in mongo (or ignoring the seconds when pinging)
@TODO: OFF/ON flag is needed to temporary disable an endpoint from pinging
*/

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/tomekwlod/ping"
	"github.com/tomekwlod/ping/db"
)

type pageResult struct {
	Page     ping.Page
	Code     int
	Duration time.Duration
	Content  string
}

var (
	cnf ping.Parameters
	p   ping.Page
	err error
)

func mgoHost() (host string) {
	// Database host from the environment variables
	host = os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost:27017"
	}

	return
}

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
	// definging the mongodb session
	mgoSession, err := db.CreateSession(mgoHost())
	defer mgoSession.Close()
	if err != nil {
		log.Panic("Cannot connect to Mongodb: ", err)
	}

	// combine the datastore session and the logger into one struct
	s := &service{
		session: mgoSession}

	// ... i do not have to open two sessions here:
	pageRepo := s.getPageRepo()
	defer pageRepo.Close()
	pageEntryRepo := s.getPageEntryRepo()
	defer pageEntryRepo.Close()

	pages, err := pageRepo.PagesForPing()
	if err != nil {
		log.Panic(err)
	}

	if len(pages) == 0 {
		log.Println("No pages found")

		return
	}

	const workers = 25

	wg := new(sync.WaitGroup)
	in := make(chan ping.Page, 2*workers)
	results := []*pageResult{}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range in {
				code, duration, _, content := urlTest(page.Url)

				results = append(results, &pageResult{page, code, duration, content})
			}
		}()
	}

	for _, page := range pages {
		in <- *page
	}

	close(in)
	wg.Wait()

	if len(results) > 0 {
		for _, row := range results {

			// sending an email only if the last status code is 200 to avoid spamming
			if (row.Page.LastStatus == 200 && row.Code != 200) || (row.Page.LastStatus != 200 && row.Code == 200) {
				sendEmail(row.Page.Url, row.Code)
			}

			// repo := repository{appC.db.C("page_entry")}

			content := ""
			if row.Code != 200 {
				content = row.Content
			}

			pageEntry := &ping.PageEntry{Code: row.Code, Load: row.Duration.Seconds(), Page: row.Page.Id}
			pageEntry.SetInsertDefaults(time.Now())

			err := pageEntryRepo.Create(pageEntry)
			if err != nil {
				log.Panicln(err)
			}

			// pageRepo := repository{appC.db.C("pages")}
			// err = pageRepo.coll.UpdateId(row.Page.Id, bson.M{"$set": bson.M{"_modified": time.Now(), "last_status": row.Code}})

			page := &row.Page
			page.LastStatus = row.Code
			page.Modified = time.Now()
			page.NextPing = time.Now().Add(time.Hour*time.Duration(0) + time.Minute*time.Duration(page.Interval) + time.Second*time.Duration(0))
			if content != "" {
				// update content only when error appears
				page.Content = content
			}

			err = pageRepo.Upsert(page)

			if err != nil {
				log.Panic(err)
			}
		}
	}
}

func urlTest(url string) (int, time.Duration, string, string) {
	if !strings.Contains(url, "http://") {
		url = "http://" + url
	}

	req, err := http.NewRequest("GET", url, nil)

	// Starting the benchmark
	timeStart := time.Now()

	resp, err := http.DefaultTransport.RoundTrip(req)

	if err != nil {
		log.Printf("%v", err)

		// How long did it take
		duration := time.Since(timeStart)

		return 404, duration, "", ""
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

func sendEmail(url string, statusCode int) {
	if cnf.SMTP_Email == "" || cnf.SMTP_Server == "" || cnf.SMTP_Port == "" || len(cnf.SMTP_Emails) == 0 {
		log.Println("SMTP credentials not set. Skipping email notification")
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
	log.Println(servername)
	// host, _, _ := net.SplitHostPort(servername)

	c, err := smtp.Dial(servername)
	if err != nil {
		log.Panic(err)
	}

	// To && From
	if err = c.Mail(cnf.SMTP_Email); err != nil {
		log.Panic(err)
	}

	for _, email := range cnf.SMTP_Emails {
		if err = c.Rcpt(email); err != nil {
			log.Panic(err)
		}
	}

	// Data
	w, err := c.Data()
	if err != nil {
		log.Panic(err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Panic(err)
	}

	err = w.Close()
	if err != nil {
		log.Panic(err)
	}

	c.Quit()

	fmt.Println("Notification sent to " + strings.Join(cnf.SMTP_Emails, ", "))
}
