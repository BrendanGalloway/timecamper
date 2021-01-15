package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type TimeCampUser struct {
	GroupID     string `json:"group_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	LoginCount  string `json:"login_count"`
	LoginTime   string `json:"login_time"`
	DisplayName string `json:"display_name"`
	SynchTime   string `json:"synch_time"`
}

type TimeCampEntry struct {
	ID               int    `json:"id"`
	Duration         string `json:"duration"`
	UserID           string `json:"user_id"`
	UserName         string `json:"user_name"`
	TaskID           string `json:"task_id"`
	LastModify       string `json:"last_modify"`
	Date             string `json:"date"`
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
	Locked           string `json:"locked"`
	Name             string `json:"name"`
	AddonsExternalID string `json:"addons_external_id"`
	Billable         int    `json:"billable"`
	InvoiceID        string `json:"invoiceId"`
	Color            string `json:"color"`
	Description      string `json:"description"`
}

func main() {

	var tcToken string
	var emailList string
	var slackhook string
	flag.StringVar(&tcToken, "tctoken", "", "Timecamp API token")
	flag.StringVar(&slackhook, "slackhook", "", "Slack message posting webhook")
	flag.StringVar(&emailList, "users", "", "comma separated list of user email to check")
	flag.Parse()
	emails := strings.Split(emailList,",")
	getUsersResponse, err := http.Get(fmt.Sprintf("https://www.timecamp.com/third_party/api/users/format/json/api_token/%v/", tcToken))
	if err != nil {
		fmt.Printf("Failed to get timecamp user list: %v", err)
		os.Exit(1)
	}
	getUsers, _ := ioutil.ReadAll(getUsersResponse.Body)
	users := []TimeCampUser{}
	json.Unmarshal(getUsers, &users)
	fulltime := make([]TimeCampUser, 0, len(emails))
	for _, email := range(emails) {
		useridfound := false
		for _, u := range(users) {
			if email == u.Email {
				fulltime = append(fulltime, u)
				useridfound = true
			}
		}
		if !useridfound {
			fmt.Sprintf("Cound not find timecamp ID for %v", email)
		}
	}

	today := time.Now()
	yesterday := today.AddDate(0,0,-1)

	naughtlyList := []string{}
	for _, u := range(fulltime) {
		getEntriesResponse, err := http.Get(fmt.Sprintf("https://www.timecamp.com/third_party/api/entries/format/json/api_token/%v/from/%v/to/%v/user_ids/%v", tcToken, yesterday.Format("2006-01-02"), today.Format("2006-01-02"), u.UserID))
		if err != nil {
			fmt.Sprintf("Failed to retrieve task list for %v: %v", u.Email, err)
			os.Exit(1)
		}
		getEntries, _ := ioutil.ReadAll(getEntriesResponse.Body)
		entries := []TimeCampEntry{}
		json.Unmarshal(getEntries, &entries)
		if len(entries) == 0 {
			naughtlyList = append(naughtlyList, strings.Title(u.Email[:strings.Index(u.Email, "@")]))
		}
	}

	var slackMsg string
	if len(naughtlyList) == 0 {
		slackMsg = "Everyone has filled in their timesheets, well done!"
	} else {
		naughtyString := strings.Join(naughtlyList, ", ")
		lastComma := strings.LastIndex(naughtyString, ",")
		if lastComma > 0 {
			naughtyString = naughtyString[:lastComma] + strings.Replace(naughtyString[lastComma:], ",", " and", 1)
			slackMsg = fmt.Sprintf("%v have not filled in their timesheets", naughtyString)
		} else {
			slackMsg = fmt.Sprintf("%v has not filled in their timesheet", naughtyString)
		}

	}
	fmt.Println(slackMsg)

	slackjson := map[string]string{"text": slackMsg}
	jsonData, _ := json.Marshal(slackjson)
	response, err := http.Post(slackhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Could not post slack message: %v", err)
		os.Exit(1)
	}
	data, _ := ioutil.ReadAll(response.Body)
	fmt.Println(string(data))
}
