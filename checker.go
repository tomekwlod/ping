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

	mgo "gopkg.in/mgo.v2"

	"github.com/tomekwlod/ping/models"
	"github.com/tomekwlod/ping/utils"
)

type pageResult struct {
	Page     models.Page
	Code     int
	Duration time.Duration
}

type repository struct {
	coll *mgo.Collection
}

type appContext struct {
	db *mgo.Database
}

func main() {
	session := utils.GetMongoSession()
	appC := appContext{session.DB(utils.DbName)}

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
				code, duration, _, _ := urlTest(page.Url)

				results = append(results, pageResult{page, code, duration})
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

			pageEntry := &models.PageEntry{Code: row.Code, Load: row.Duration.Seconds(), Page: row.Page.Id}
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

func sendEmail(url string, statusCode int) {
	// username and password \n seperated
	authFile, err := ioutil.ReadFile("auth.txt")

	if err != nil {
		log.Fatal(err)
	}

	credentials := strings.Split(string(authFile), "\n")

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

		body = "Hi Phase II team member,\n\n" +
			"This is a notification sent by Ping®.\n\n" +

			"Incident (" + message + ") for `" + url + "`, has been assigned to you.\n\n" +
			"You will be notified when the page goes live back again.\n\n" +

			"Best regards,\n" +
			"The Phase II Team.\n" +
			"Tomek Wlodarczyk\r\n"
	} else {
		subject = "Subject: Incident CLOSED for " + url + " "
		body = "Hi Phase II team member,\n\n" +
			"This is a notification sent by Ping®.\n\n" +

			"Incident CLOSED for `" + url + "`\n\n" +

			"Best regards,\n" +
			"The Phase II Team.\n" +
			"Tomek Wlodarczyk\r\n"
	}

	smtpSrv := "smtp.gmail.com"
	to := []string{"twl@phase-ii.com"}
	msg := []byte("To: " + strings.Join(to, ", ") + "\r\n" +
		subject + " !!\r\n" +
		"\r\n" + body)

	// sometimes this needs to be clicked to add your not-secured device
	//  https://accounts.google.com/DisplayUnlockCaptcha
	auth := smtp.PlainAuth("",
		credentials[0],
		credentials[1],
		smtpSrv,
	)

	err = smtp.SendMail(
		smtpSrv+":587",
		auth,
		credentials[0],
		to,
		msg,
	)

	if err != nil {
		log.Print("ERROR: attempting to send a mail ", err)
	}

	fmt.Println("Notification sent to " + strings.Join(to, ", "))
}

func pages(session *mgo.Session) models.PageCollection {
	result := models.PageCollection{[]models.Page{}}

	appC := appContext{session.DB(utils.DbName)}
	repo := repository{appC.db.C("pages")}

	err := repo.coll.Find(nil).All(&result.Data)

	if err != nil {
		log.Fatal(err)
	}

	return result
}
