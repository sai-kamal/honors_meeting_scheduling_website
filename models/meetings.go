package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/schema"
)

var (
	//ChatServers contains all the different chatroom servers
	ChatServers map[int64]*Server
)

//Meeting struct defines the necessary parameters related to the user
type Meeting struct {
	MTime          string `json:"time" schema:"time"` //start time of meeting
	Name           int64  `json:"name" schema:"name"` //name of meeting
	NumAttendees   int64  `json:"num_attendees" schema:"num_attendees"`
	TimeSpace      int64  `json:"time_space" schema:"time_space"`
	TimeDiff       int64  `json:"time_diff" schema:"time_diff"`
	ActionTimeDiff int64  `json:"action_time_diff" schema:"action_time_diff"`
	NoCntrlEnts    int64  `json:"no_cntrl_ents" schema:"no_cntrl_ents"`
	IsComplete     bool   `json:"is_complete" schema:"is_complete"`
}

//MeetingsInit initializes everything for the meetings
func MeetingsInit() {
	ChatServers = make(map[int64]*Server)
}

//GetMeetings gets the meeting info from the database
func GetMeetings(route string) []Meeting {
	var meetings []Meeting
	queryStr := fmt.Sprintf("SELECT * FROM meetings")
	resp, qErr := DBquery(queryStr)
	if qErr != nil || len(resp[0].Series) == 0 {
		log.Println("failed to get meetings data from DB in ", route, qErr)
		return meetings
	}
	for _, val := range resp[0].Series[0].Values {
		var meeting Meeting
		meeting.MTime = val[0].(string)
		meeting.ActionTimeDiff, _ = val[1].(json.Number).Int64()
		meeting.IsComplete = val[2].(bool)
		meeting.Name, _ = val[3].(json.Number).Int64()
		meeting.NoCntrlEnts, _ = val[4].(json.Number).Int64()
		meeting.NumAttendees, _ = val[5].(json.Number).Int64()
		meeting.TimeDiff, _ = val[6].(json.Number).Int64()
		meeting.TimeSpace, _ = val[7].(json.Number).Int64()
		meetings = append(meetings, meeting)
	}
	return meetings
}

//CreateMeetingHandler creates and adds a meeting to the database
func CreateMeetingHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("failed to parse form during create meeting")
	}
	var meeting Meeting
	decoder := schema.NewDecoder()
	err = decoder.Decode(&meeting, r.PostForm)
	if err != nil {
		log.Println("failed to parse form from client in createMeeting", err)
	}

	url := "http://localhost:8000/policy/make_policy/"
	jsonData := map[string]int64{
		"name":             int64(meeting.Name),
		"num_attendees":    int64(meeting.NumAttendees),
		"time_space":       int64(meeting.TimeSpace),
		"time_diff":        int64(meeting.TimeDiff),
		"action_time_diff": int64(meeting.ActionTimeDiff),
		"no_cntrl_ents":    int64(meeting.NoCntrlEnts),
	}
	jsonDataBytes, _ := json.Marshal(jsonData)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonDataBytes))
	if err != nil {
		log.Println("failed to run make_policy", err)
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to get body from request in CreateMeetingHandler", err)
	}
	defer resp.Body.Close()
	//initialize chat room server
	server := NewServer(meeting)
	ChatServers[meeting.Name] = server
	go server.Listen() // listens to all the requests to the server room

	measurement := "meetings"
	tags := map[string]string{}
	fields := map[string]interface{}{
		"name":             float64(meeting.Name),
		"num_attendees":    float64(meeting.NumAttendees),
		"time_space":       float64(meeting.TimeSpace),
		"time_diff":        float64(meeting.TimeDiff),
		"action_time_diff": float64(meeting.ActionTimeDiff),
		"no_cntrl_ents":    float64(meeting.NoCntrlEnts),
		"is_complete":      false,
	}
	t := time.Now()
	DBwrite(measurement, tags, fields, t)
	http.Redirect(w, r, "/meetings", http.StatusSeeOther)
	return
}
