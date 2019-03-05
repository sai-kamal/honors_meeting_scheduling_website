package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

const (
	//PythonServerURL describes the URL of the python server
	PythonServerURL string = "http://localhost:8000/policy/"
)

var (
	//ChatServers contains all the different chatroom servers
	ChatServers map[int64]*Server
)

//Meeting struct defines the necessary parameters related to the user
type Meeting struct {
	DBTime         string `json:"time,omitempty" schema:"time,omitempty"` //time the meeting was created in DB
	Name           int64  `json:"name,omitempty" schema:"name,omitempty"` //name of meeting
	NumAttendees   int64  `json:"num_attendees,omitempty" schema:"num_attendees,omitempty"`
	TimeSpace      int64  `json:"time_space,omitempty" schema:"time_space,omitempty"`
	TimeDiff       int64  `json:"time_diff,omitempty" schema:"time_diff,omitempty"`
	ActionTimeDiff int64  `json:"action_time_diff,omitempty" schema:"action_time_diff,omitempty"`
	NoCntrlEnts    int64  `json:"no_cntrl_ents,omitempty" schema:"no_cntrl_ents,omitempty"`
	CurrExpect     int64  `json:"current_expect,omitempty" schema:"current_expect,omitempty"`
	OrigExpect     int64  `json:"orig_expect,omitempty" schema:"orig_expect,omitempty"`
	IsComplete     bool   `json:"is_complete,omitempty" schema:"is_complete,omitempty"`
	Feedback       int64  `json:"feedback,omitempty" schema:"feedback,omitempty"`
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
		meeting.DBTime = val[0].(string)
		meeting.ActionTimeDiff, _ = val[1].(json.Number).Int64()
		meeting.IsComplete = val[2].(bool)
		meeting.Name, _ = val[3].(json.Number).Int64()
		meeting.NoCntrlEnts, _ = val[4].(json.Number).Int64()
		meeting.NumAttendees, _ = val[5].(json.Number).Int64()
		meeting.OrigExpect, _ = val[6].(json.Number).Int64()
		meeting.TimeDiff, _ = val[7].(json.Number).Int64()
		meeting.TimeSpace, _ = val[8].(json.Number).Int64()
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

	meeting.CurrExpect = meeting.OrigExpect

	jsonData := map[string]int64{
		"name":             int64(meeting.Name),
		"num_attendees":    int64(meeting.NumAttendees),
		"time_space":       int64(meeting.TimeSpace),
		"time_diff":        int64(meeting.TimeDiff),
		"action_time_diff": int64(meeting.ActionTimeDiff),
		"no_cntrl_ents":    int64(meeting.NoCntrlEnts),
		"orig_expect":      int64(meeting.OrigExpect),
	}
	jsonDataBytes, _ := json.Marshal(jsonData)
	resp, err := http.Post(PythonServerURL+"make_policy/", "application/json", bytes.NewBuffer(jsonDataBytes))
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
	// go server.Listen() // listens to all the requests to the server room
	measurement := "meetings"
	tags := map[string]string{}
	fields := map[string]interface{}{
		"name":             float64(meeting.Name),
		"num_attendees":    float64(meeting.NumAttendees),
		"time_space":       float64(meeting.TimeSpace),
		"time_diff":        float64(meeting.TimeDiff),
		"action_time_diff": float64(meeting.ActionTimeDiff),
		"no_cntrl_ents":    float64(meeting.NoCntrlEnts),
		"orig_expect":      float64(meeting.OrigExpect),
		"is_complete":      false,
	}
	t := time.Now()
	DBwrite(measurement, tags, fields, t)
	http.Redirect(w, r, "/meetings", http.StatusSeeOther)
	return
}

// StartMeetingHandler starts the chat server listening functionality
//used to control the number of people in a room and have them to wait until everything is set
func StartMeetingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	meetingName, err := strconv.Atoi(vars["meetingName"])
	if err != nil {
		log.Println("failed to convert string to int64", err)
	}
	go ChatServers[int64(meetingName)].Listen()
	return
	//TODO:make superuser/admin to see the log room
}
