package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const channelBufSize = 100

//UserMeetingParams contains info about users who joined a meeting
type UserMeetingParams struct {
	MeetingName int64  `json:"meeting_name" schema:"meeting_name"`
	Username    string `json:"username" schema:"username"`
	Delay       int64  `json:"delay" schema:"delay"` //agent expectation
	// CntrlEnt         int64  `json:"cntrl_ent" schema:"cntrl_ent"`
	Importance        int64 `json:"importance" schema:"importance"` // to represent the meeting importance to the user
	Conn              *websocket.Conn
	Server            *Server
	OutgoingMessageCh chan *Message
	DoneCh            chan bool
	ActionCh          chan ActionInfo
	Cond              *sync.Cond
	sync.Mutex
}

//ActionQuery is used in requests to find out the action in the mdp policy at a states
type ActionQuery struct {
	CntrlEntity   int64 `json:"cntrl_ent" schema:"cntrl_ent"`
	CurrentExpect int64 `json:"tcer" schema:"tcer"`
	AgentExpect   int64 `json:"aer" schema:"aer"`
	MeetingStatus int64 `json:"a_status" schema:"a_status"`
	Response      int64 `json:"resp" schema:"resp"`
	Time          int64 `json:"time" schema:"time"`
	Importance    int64 `json:"imp" schema:"imp"`
	MeetingName   int64 `json:"meeting_name" schema:"meeting_name"`
}

//ActionInfo is used to store unmarshalled data about a single action from python server
type ActionInfo struct {
	Action     int64  `json:"action"`
	ActionName string `json:"action_name"`
}

//ActionArrays is used to store 2 arrays for a entity from python server
type ActionArrays struct {
	Actions      []float64 `json:"actions"`
	ActionsNames []string  `json:"actions_names"`
}

// CreateUserMeetingParams creates a new UserMeetingParams
func CreateUserMeetingParams(conn *websocket.Conn, server *Server, user *UserMeetingParams) {
	if conn == nil {
		log.Println("connection is nil for", user)
	}
	if server == nil {
		log.Println(" Server is nil for", user)
	}
	user.Conn = conn
	user.Server = server
	user.OutgoingMessageCh = make(chan *Message, channelBufSize) //messages byy a user can be multiple
	user.DoneCh = make(chan bool, 10)
	user.Cond = sync.NewCond(user) //created condition variable for user
	user.ActionCh = make(chan ActionInfo)
	return
}

//Write sends the user a message
func (user *UserMeetingParams) Write(message Message) {
	select {
	case user.OutgoingMessageCh <- &message:
	default:
		user.Server.RemoveUser(user)
		err := fmt.Errorf("userMeetingParams %s is disconnected", user.Username)
		log.Println("err in user.Write", err)
	}
}

//Done signals the user that it is done
func (user *UserMeetingParams) Done() {
	user.DoneCh <- true
	// user.Conn.Close()
	var message Message
	message.Type = "close_connection"
	user.Conn.WriteJSON(&message)
	user.Cond.Signal()
	fmt.Println("user.Done called")
}

//InitTimes initializes all times as soon as server starts to listen in browser for all users
func (user *UserMeetingParams) InitTimes() {
	//changes origExpect, agentExpect/delay and curr_expect on screen for all users
	var msg Message
	msg.Type = "change_time"
	msg.Time = strconv.Itoa(int(user.Server.Time*user.Server.MeetingParams.TimeDiff)) + " min"
	user.Write(msg)
	msg.Type = "change_orig_expect"
	msg.Time = strconv.Itoa(int(user.Server.MeetingParams.OrigExpect*user.Server.MeetingParams.TimeDiff)) + " min"
	user.Write(msg)
	msg.Type = "change_curr_expect"
	msg.Time = strconv.Itoa(int(user.Server.MeetingParams.CurrExpect*user.Server.MeetingParams.TimeDiff)) + " min"
	user.Write(msg)
	user.ChangeDelayDisp()
}

//ChangeDelayDisp is used to change the delay in the browser
func (user *UserMeetingParams) ChangeDelayDisp() {
	var msg Message
	msg.Type = "change_agent_expect"
	msg.Time = strconv.Itoa(int(user.Delay*user.Server.MeetingParams.TimeDiff)) + " min"
	user.Write(msg)
}

//ChangeDelayDB updates DB to register the updated delay of the user.
func (user *UserMeetingParams) ChangeDelayDB() {
	measurement := strconv.Itoa(int(user.Server.MeetingParams.Name))
	tags := map[string]string{
		"type": "changed_user_delay",
	}
	fields := map[string]interface{}{
		"username": user.Username,
		"delay_to": user.Delay,
		"time":     user.Server.Time,
	}
	t := time.Now()
	DBwrite(measurement, tags, fields, t)
}

//AddActionInfoToDB updates DB to register the updated delay of the user.
func (user *UserMeetingParams) AddActionInfoToDB(action ActionInfo, who string) {
	measurement := strconv.Itoa(int(user.Server.MeetingParams.Name))
	tags := map[string]string{
		"type": "action",
	}
	fields := map[string]interface{}{
		"done_by":    who,
		"username":   user.Username,
		"action_ind": action.Action,
		"action":     action.ActionName,
		"time":       user.Server.Time,
	}
	t := time.Now()
	DBwrite(measurement, tags, fields, t)
}

//Listen writes and reads for the user
func (user *UserMeetingParams) Listen() {
	go user.AgentWorks()
	go user.listenWrite()
	user.listenRead()
}

func (user *UserMeetingParams) listenWrite() {
	for {
		select {
		//send message to user from server
		case msg := <-user.OutgoingMessageCh:
			user.Conn.WriteJSON(&msg)
		case <-user.DoneCh:
			user.DoneCh <- true
			return
		}
	}
}

func (user *UserMeetingParams) listenRead() {
	for {
		select {
		//receive Done request
		case <-user.DoneCh:
			user.DoneCh <- true
			return
		// read data from websocket connection
		default:
			var message Message
			err := user.Conn.ReadJSON(&message)
			fmt.Println("message type", message.Type)
			if err != nil {
				user.DoneCh <- true
				// go user.Done()
				log.Println("Error while reading JSON from websocket ", err.Error())
			} else {
				if message.Type == "close_connection" {
					return
				} else if message.Type == "change_user_delay" { //gets message from client to change user delay
					user.Lock() //synchronizing delay for user
					intTime, err := strconv.Atoi(message.Time)
					if err != nil {
						log.Println("failed to convert string to int while changing delay of user", user.Username, err)
					}
					user.Delay = int64(intTime)
					user.ChangeDelayDB()
					user.ChangeDelayDisp()
					user.Unlock()
					user.Cond.Signal() //signalling user as his delay has changed
				} else if message.Type == "transfer_control_reply" {
					user.ActionCh <- ActionInfo{
						Action:     message.ActionInt,
						ActionName: message.Action,
					}
				}
			}
		}
	}
}

//AgentWorks runs the agent and mimics actions in the background
func (user *UserMeetingParams) AgentWorks() {
	//just for testing transfer_control in user
	// actions := user.GetActions(1)
	// var msg Message
	// msg.Type = "transfer_control"
	// msg.Actions = actions.Actions
	// msg.ActionsNames = actions.ActionsNames
	// user.Write(msg)

	for {
		select {
		case <-user.DoneCh:
			user.DoneCh <- true
			return
		default:
			user.Lock()
			user.Cond.Wait()
			user.Server.Lock()
			//safe to look upon TCER and AER(delays)
			meeting := &user.Server.MeetingParams
			aer := user.Delay
			tcer := meeting.CurrExpect
			if aer-tcer > 0 { //user is still not in the meeting
				var query ActionQuery //signifies state
				query.CntrlEntity = 0 //agent
				query.CurrentExpect = tcer
				query.AgentExpect = aer
				query.MeetingStatus = 0
				query.Response = 1
				query.Time = user.Server.Time
				query.Importance = user.Importance - 1
				query.MeetingName = meeting.Name

				actionInfo := user.getSingleAction(query)
				user.AddActionInfoToDB(actionInfo, "agent") //records in database that agent does this action
				ret := user.ChangeParamsWithAction(actionInfo)

				user.Server.Unlock()
				user.Unlock()
				if ret != 0 { //only signal users when there has been a change in the params
					user.Server.SignalUsers()
				}
			} else {
				user.Server.Unlock()
				user.Unlock()
			}
		}
	}
}

//getSingleAction gets the action for a state from the python server
func (user *UserMeetingParams) getSingleAction(query ActionQuery) ActionInfo {
	jsonQuery, err := json.Marshal(query)
	if err != nil {
		log.Println("failed to convert query to JSON form", err)
	}
	resp, err := http.Post(PythonServerURL+"get_action/", "application/json", bytes.NewBuffer(jsonQuery))
	if err != nil {
		log.Println("failed to read resp in agentworks", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read body from response<-AgentWorks", err)
	}
	var actionInfo ActionInfo
	err = json.Unmarshal(body, &actionInfo)
	if err != nil {
		log.Println("failed to unmarshal resp into actionInfo<-AgentWorks", err)
	}
	return actionInfo
}

//GetActions gets the action for a state from the python server
func (user *UserMeetingParams) GetActions(who int) ActionArrays { //who == 1 is user else it is agent if who == 0
	query := ActionQuery{
		MeetingName: user.Server.MeetingParams.Name,
	}
	jsonQuery, err := json.Marshal(query)
	if err != nil {
		log.Println("failed to convert query to JSON form", err)
	}
	var url string
	if who == 0 { //agent
		url = PythonServerURL + "get_agent_actions/"
	} else if who == 1 { //user
		url = PythonServerURL + "get_user_actions/"
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonQuery))
	if err != nil {
		log.Println("failed to read resp in get Actions", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read body from response AgentWorks", err)
	}
	var actionArrays ActionArrays
	err = json.Unmarshal(body, &actionArrays)
	if err != nil {
		log.Println("failed to unmarshal resp into actionArrays getAction", err)
	}
	fmt.Println(actionArrays)
	return actionArrays
}

// ChangeParamsWithAction changes meeting params according to action
func (user *UserMeetingParams) ChangeParamsWithAction(actionInfo ActionInfo) int {
	meeting := &user.Server.MeetingParams
	action := actionInfo.Action
	fmt.Println("action is", actionInfo.ActionName)
	ret := 1 //acion is not "no action"
	if action == 0 {
		log.Println("action is to do nothing, so DID NOTHING")
		ret = 0
	} else if action >= 1 && action <= int64((meeting.TimeSpace-(meeting.OrigExpect*meeting.TimeDiff))/meeting.ActionTimeDiff) {
		if user.Server.Time*meeting.TimeDiff < (action*meeting.ActionTimeDiff)+(meeting.OrigExpect*meeting.TimeDiff) {
			meeting.CurrExpect = int64(meeting.OrigExpect + int64(action*(meeting.ActionTimeDiff/meeting.TimeDiff)))
			user.Server.AddCurrExpectInfoToDB()
			user.Server.ChangeCurrExpectDisp() //changes curr_expect for all the users in the browseer
		}
	} else if actionInfo.ActionName == "transfer_control" {
		actions := user.GetActions(1)
		var msg Message
		msg.Type = "transfer_control"
		msg.Actions, msg.ActionsNames = actions.Actions, actions.ActionsNames
		user.Write(msg)
		act := <-user.ActionCh
		fmt.Println("action chosen by user is ", act)
		user.AddActionInfoToDB(act, "user") //records action into DB
		ret = user.ChangeParamsWithAction(act)
		// do analysis and change params accordingly
	} else if actionInfo.ActionName == "cancel" {
		for _, u := range user.Server.ConnectedUsers {
			u.Done()
		}
	} else if actionInfo.ActionName == "arrive_at_meet" {
		user.Done()
	} else if actionInfo.ActionName == "will_not_attend" {
		user.Done()
	} else {
		log.Println("action not recognized")
	}
	return ret
}
