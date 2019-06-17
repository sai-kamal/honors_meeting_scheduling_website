package models

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

//Message struct used to send messages b/n server and client
type Message struct {
	Type         string    `json:"type"`
	Time         string    `json:"time"`
	Username     string    `json:"username"`
	Message      string    `json:"message"`
	ActionInt    int64     `json:"action_int"`
	Action       string    `json:"action"`
	Sender       int64     `json:"sender"` //agent or user
	Actions      []float64 `json:"actions"`
	ActionsNames []string  `json:"actions_names"`
}

//Server is the chatroom instance
type Server struct {
	ConnectedUsers       map[string]*UserMeetingParams // string stores the username of the user
	Messages             []Message
	AddUserCh            chan *UserMeetingParams
	RemoveUserCh         chan *UserMeetingParams
	NewIncomingMessageCh chan Message
	DoneCh               chan bool
	Cond                 *sync.Cond
	Time                 int64
	MeetingParams        Meeting //contains info about the meeting
	sync.Mutex
}

//MeetingsInit initializes everything for the meetings
func MeetingsInit() {
	ChatServers = make(map[int64]*Server)
}

//NewServer creates a new server for the chatroom
func NewServer(meeting Meeting) *Server {
	var server Server
	server.ConnectedUsers = make(map[string]*UserMeetingParams)
	server.Messages = []Message{}
	server.AddUserCh = make(chan *UserMeetingParams)
	server.RemoveUserCh = make(chan *UserMeetingParams)
	server.NewIncomingMessageCh = make(chan Message)
	server.DoneCh = make(chan bool, 10)
	server.MeetingParams = meeting
	server.Cond = sync.NewCond(&server)
	server.Time = 0
	return &server
}

//SignalUsers signals the condition variable for all the users connected to server.
func (server *Server) SignalUsers() {
	for _, user := range server.ConnectedUsers {
		user.Cond.Signal()
	}
}

//AddUser adds a new user to the server
func (server *Server) AddUser(user *UserMeetingParams) {
	server.AddUserCh <- user
	return
}

//RemoveUser removes a new user from the server
func (server *Server) RemoveUser(user *UserMeetingParams) {
	server.RemoveUserCh <- user
}

//Done signals that it is done
func (server *Server) Done() {
	server.DoneCh <- true
}

//SendAll sends the message to all the users present in the chatroom
func (server *Server) SendAll(msg Message) {
	for _, user := range server.ConnectedUsers {
		user.Write(msg)
	}
}

//ChangeTime sends message(through socket) to all users to change time of meeting in browser
func (server *Server) ChangeTime() {
	var msg Message //changes time on screen for all users
	msg.Type = "change_time"
	msg.Time = strconv.Itoa(int(server.Time*server.MeetingParams.TimeDiff)) + " min"
	server.SendAll(msg)
}

//ChangeCurrExpectDisp sends message(through socket) to all users to change curr_expect
func (server *Server) ChangeCurrExpectDisp() {
	var msg Message //changes time on screen for all users
	msg.Type = "change_curr_expect"
	msg.Time = strconv.Itoa(int(server.MeetingParams.CurrExpect*server.MeetingParams.TimeDiff)) + " min"
	server.SendAll(msg)
}

//Listen listens and responds to requests in the chatroom
func (server *Server) Listen() {
	log.Println("chat server, ", server.MeetingParams.Name, " Listening .....")
	ticker := time.NewTicker(25 * time.Second)
	for {
		fmt.Println("server time is ", server.Time)
		select {
		// Adding a new user
		case user := <-server.AddUserCh:
			log.Println("Added a new User to the room", user)
			server.ConnectedUsers[user.Username] = user
			server.AddUserInfoToDB("add", "user "+user.Username+" added")
			user.InitTimes()
			user.Lock()
			user.ChangeDelayDB()
			user.Unlock()
			user.Cond.Signal()
		//removing a new user
		case user := <-server.RemoveUserCh:
			log.Println("Removing user from chat room")
			delete(server.ConnectedUsers, user.Username)
			server.AddUserInfoToDB("remove", "user "+user.Username+" removed")
			user.Done()
		// change meeting time every 10 sec with the next timesptamp. Should change tcer, time and should signal all the users.
		case <-ticker.C:
			server.Time++
			//checks if the current time > max allowed time, then remove and close every thing
			if server.Time > (server.MeetingParams.TimeSpace / server.MeetingParams.TimeDiff) {
				server.Done()
				// server.DoneCh <- true
				fmt.Println("donech has true now")
			} else {
				server.ChangeTime()
				server.CheckTimeAndCurrExpect()
			}
		case <-server.DoneCh:
			ticker.Stop()
			server.CloseEverything()
			return
		}
	}
}

//AddUserInfoToDB records in db the action
func (server *Server) AddUserInfoToDB(typeAction string, action interface{}) {
	measurement := strconv.Itoa(int(server.MeetingParams.Name))
	tags := map[string]string{
		"type": typeAction,
	}
	fields := map[string]interface{}{
		"action": action,
		"time":   server.Time,
	}
	t := time.Now()
	DBwrite(measurement, tags, fields, t)
}

// CheckTimeAndCurrExpect checks and updates time to match with curr expect
func (server *Server) CheckTimeAndCurrExpect() {
	server.Lock()
	if server.Time > server.MeetingParams.CurrExpect {
		server.MeetingParams.CurrExpect = server.Time
		server.AddCurrExpectInfoToDB() //update DB with the new curr_expect
		server.SignalUsers()           //signalling all the users that the curr_expect has changed
		server.ChangeCurrExpectDisp()
	}
	server.Unlock()
}

//AddCurrExpectInfoToDB records in db the action
func (server *Server) AddCurrExpectInfoToDB() {
	measurement := strconv.Itoa(int(server.MeetingParams.Name))
	tags := map[string]string{
		"type": "changed_curr_expect",
	}
	fields := map[string]interface{}{
		"curr_expect": server.MeetingParams.CurrExpect,
		"time":        server.Time,
	}
	t := time.Now()
	DBwrite(measurement, tags, fields, t)
}

//CloseEverything shutdowns everything related to the server
func (server *Server) CloseEverything() {
	for _, user := range server.ConnectedUsers { //signal and close every user connection
		//removing users connected to server
		log.Println("Removing user from chat room")
		delete(server.ConnectedUsers, user.Username)
		server.AddUserInfoToDB("remove", "user "+user.Username+" removed")
		user.Done()
	}

	//removing mdp and params from python server's memory
	resp, err := http.Get(PythonServerURL + "delete_policy/" + strconv.Itoa(int(server.MeetingParams.Name)))
	if err != nil {
		log.Println("failed to clos mdp in python server", err)
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to get body from resp delete_policy from python server", err)
	}
	resp.Body.Close()

	// remove meeting from current meetings and stop its own server in go
	delete(ChatServers, server.MeetingParams.Name)
	log.Println("deleted chat server " + strconv.Itoa(int(server.MeetingParams.Name)) + " from ChatServers map")
}
