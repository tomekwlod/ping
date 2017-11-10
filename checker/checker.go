package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/tomekwlod/ping/config"
	"github.com/tomekwlod/ping/models"
	"github.com/tomekwlod/ping/utils"
)

type pageResult struct {
	Page     models.Page
	Code     int
	Duration time.Duration
	Content  string
}

type repository struct {
	coll *mgo.Collection
}

type appContext struct {
	db *mgo.Database
}

func main() {
	session := utils.MongoSession()
	appC := appContext{session.DB(config.Params.DB.DbName)}

	results := []pageResult{}
	pages := pages(session)

	if len(pages.Data) == 0 {
		log.Println("No pages found")
		return
	}

	const workers = 25

	wg := new(sync.WaitGroup)
	in := make(chan models.Page, 2*workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range in {
				code, duration, _, content := urlTest(page.Url)

				results = append(results, pageResult{page, code, duration, content})
			}
		}()
	}

	for _, page := range pages.Data {
		in <- page
	}

	close(in)
	wg.Wait()

	if len(results) > 0 {
		for _, row := range results {

			// sending an email only if the last status code is 200 to avoid spamming
			if (row.Page.LastStatus == 200 && row.Code != 200) || (row.Page.LastStatus != 200 && row.Code == 200) {
				sendEmail(row.Page.Url, row.Code)
			}

			repo := repository{appC.db.C("page_entry")}

			content := ""
			if row.Code != 200 {
				content = row.Content
			}

			pageEntry := &models.PageEntry{Code: row.Code, Load: row.Duration.Seconds(), Page: row.Page.Id, Content: content}
			pageEntry.SetInsertDefaults(time.Now())

			err := repo.coll.Insert(pageEntry)
			if err != nil {
				log.Panic(err)
			}

			pageRepo := repository{appC.db.C("pages")}
			// err = pageRepo.coll.UpdateId(row.Page.Id, bson.M{"$set": bson.M{"_modified": time.Now(), "last_status": row.Code}})

			page := &row.Page
			page.LastStatus = row.Code
			page.Modified = time.Now()
			pageRepo.coll.UpsertId(row.Page.Id, page)

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
	config := config.Params

	if config.SMTP.Email == "" || config.SMTP.Password == "" || config.SMTP.Server == "" || config.SMTP.Port == "" || len(config.SMTP.Emails) == 0 {
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
		subject = "Subject: Incident OPEN (" + message + ") for " + url + " "

		body = "Hi there,\n\n" +
			"This is a notification sent by Ping®.\n\n" +

			"Incident (" + message + ") for `" + url + "`, has been assigned to you.\n\n" +
			"You will be notified when the page goes live back again.\n\n" +

			"Best regards,\n" +
			"Ping®\r\n"
	} else {
		subject = "Subject: Incident CLOSED for " + url + " "
		body = "Hi there,\n\n" +
			"This is a notification sent by Ping®.\n\n" +

			"Incident CLOSED for `" + url + "`\n\n" +

			"Best regards,\n" +
			"Ping®\r\n"
	}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = config.SMTP.Email
	headers["To"] = strings.Join(config.SMTP.Emails, ",")
	headers["Subject"] = subject

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server
	servername := config.SMTP.Server + ":" + config.SMTP.Port
	log.Println(servername)
	host, _, _ := net.SplitHostPort(servername)

	auth := smtp.PlainAuth("", config.SMTP.Email, config.SMTP.Password, host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	// Here is the key, you need to call tls.Dial instead of smtp.Dial
	// for smtp servers running on 465 that require an ssl connection
	// from the very beginning (no starttls)
	conn, err := tls.Dial("tcp", servername, tlsconfig)
	if err != nil {
		log.Panic(err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Panic(err)
	}

	// Auth
	if err = c.Auth(auth); err != nil {
		log.Panic(err)
	}

	// To && From
	if err = c.Mail(config.SMTP.Email); err != nil {
		log.Panic(err)
	}

	for _, email := range config.SMTP.Emails {
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

	fmt.Println("Notification sent to " + strings.Join(config.SMTP.Emails, ", "))
}

func pages(session *mgo.Session) models.PageCollection {
	result := models.PageCollection{[]models.Page{}}

	appC := appContext{session.DB(config.Params.DB.DbName)}
	repo := repository{appC.db.C("pages")}

	err := repo.coll.Find(nil).All(&result.Data)

	if err != nil {
		log.Fatal(err)
	}

	return result
}
