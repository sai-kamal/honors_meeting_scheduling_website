package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

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
	Cond              *sync.Cond
	sync.Mutex
}

//ActionQuery is used in requests to find out the action in the mdp policy
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
	user.DoneCh = make(chan bool)
	user.Cond = sync.NewCond(user) //created condition variable for user
	return
}

//Write sends the user a message
func (user *UserMeetingParams) Write(message Message) {
	select {
	case user.OutgoingMessageCh <- &message:
	default:
		user.Server.RemoveUser(user)
		err := fmt.Errorf("userMeetingParams %s is disconnected", user.Username)
		log.Println("err in Write of user", err)
	}
}

//Done signals the user that it is done
func (user *UserMeetingParams) Done() {
	user.DoneCh <- true
}

//Listen writes and reads for the user
func (user *UserMeetingParams) Listen() {
	go user.listenWrite()
	go user.AgentWorks()
	user.listenRead()
}

func (user *UserMeetingParams) listenWrite() {
	for {
		select {
		//send message to user from server
		case msg := <-user.OutgoingMessageCh:
			user.Conn.WriteJSON(&msg)
		case <-user.DoneCh:
			user.Server.RemoveUser(user)
			user.DoneCh <- true
			return
		}
	}
}

func (user *UserMeetingParams) listenRead() {
	//log.Println("Listening to Read to client")
	for {
		select {
		//receive Done request
		case <-user.DoneCh:
			user.Server.RemoveUser(user)
			user.DoneCh <- true
			return
		// read data from websocket connection
		default:
			var messageObj Message
			err := user.Conn.ReadJSON(&messageObj)
			if err != nil {
				user.DoneCh <- true
				log.Println("Error while reading JSON from websocket ", err.Error())
			} else {
				user.Server.ProcessNewIncomingMessage(messageObj)
			}
		}
	}
}

//AgentWorks runs the agent and mimics actions in the background
func (user *UserMeetingParams) AgentWorks() {
	for {
		user.Lock()
		user.Cond.Wait()
		user.Server.Lock()
		//safe to look upon TCER and AER(delays)
		meeting := &user.Server.MeetingParams
		aer := user.Delay
		tcer := meeting.CurrExpect
		fmt.Println("aer, tcer", aer, tcer)
		if aer-tcer > 0 { //user is still not in the meeting
			var query ActionQuery
			query.CntrlEntity = 0 //agent
			query.CurrentExpect = tcer
			query.AgentExpect = aer
			query.MeetingStatus = 0
			query.Response = 1
			query.Time = user.Server.Time
			query.Importance = user.Importance - 1
			query.MeetingName = meeting.Name

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
			var actionInfo struct {
				Action     int64  `json:"action"`
				ActionName string `json:"action_name"`
			}
			err = json.Unmarshal(body, &actionInfo)
			if err != nil {
				log.Println("failed to unmarshal resp into actionInfo<-AgentWorks", err)
			}
			action := actionInfo.Action
			//changing parameters as per action
			if action >= 1 && action <= int64((meeting.TimeSpace-(meeting.OrigExpect*meeting.TimeDiff))/meeting.ActionTimeDiff) {
				if user.Server.Time*meeting.TimeDiff < (action*meeting.ActionTimeDiff)+(meeting.OrigExpect*meeting.TimeDiff) {
					meeting.CurrExpect = int64(meeting.OrigExpect + int64(action*(meeting.ActionTimeDiff/meeting.TimeDiff)))
				}
			} else if actionInfo.ActionName == "transfer_control" {
				//TODO
				fmt.Println("ask user for action with a time limit")
			}
			user.Server.Unlock()
			user.Unlock()
			user.Server.SignalUsers()
		} else {
			user.Server.Unlock()
			user.Unlock()
		}
	}
}

//ChangeDelayHandler changes the delay of the user in the meeting
// func ChangeDelayHandler(w http.ResponseWriter, r *http.Request) {

// }
