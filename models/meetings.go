package models

import (
	"encoding/json"
	"fmt"
	"log"
)

//Meeting struct defines the necessary parameters related to the user
type Meeting struct {
	MTime          string `json:"time" schema:"time"`
	Name           int64  `json:"name" schema:"name"`
	NumAttendees   int64  `json:"num_attendees" schema:"num_attendees"`
	TimeSpace      int64  `json:"time_space" schema:"time_space"`
	TimeDiff       int64  `json:"time_diff" schema:"time_diff"`
	ActionTimeDiff int64  `json:"action_time_diff" schema:"action_time_diff"`
	NoCntrlEnts    int64  `json:"no_cntrl_ents" schema:"no_cntrl_ents"`
	IsComplete     bool   `json:"is_complete" schema:"is_complete"`
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
