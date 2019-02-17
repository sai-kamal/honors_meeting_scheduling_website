package models

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

//Message struct used to
//TODO: create correct message struct
type Message struct {
	Username  string `json:"username"`
	Message   string `json:"message"`
	ActionInt int    `json:"action_int"`
	Action    string `json:"action"`
	Sender    int    `json:"sender"` //agent or user
}

//Server is the chatroom instance
type Server struct {
	ConnectedUsers       map[string]*UserMeetingParams // string stores the username of the user
	Messages             []Message
	AddUserCh            chan *UserMeetingParams
	RemoveUserCh         chan *UserMeetingParams
	NewIncomingMessageCh chan Message
	ErrorCh              chan error
	DoneCh               chan bool
	MeetingParams        Meeting //contains info about the meeting
}

//NewServer creates a new server for the chatroom
func NewServer(meeting Meeting) *Server {
	ConnectedUsers := make(map[string]*UserMeetingParams)
	Messages := []Message{}
	AddUserCh := make(chan *UserMeetingParams)
	RemoveUserCh := make(chan *UserMeetingParams)
	NewIncomingMessage := make(chan Message)
	ErrorCh := make(chan error)
	DoneCh := make(chan bool)

	return &Server{
		ConnectedUsers,
		Messages,
		AddUserCh,
		RemoveUserCh,
		NewIncomingMessage,
		ErrorCh,
		DoneCh,
		meeting,
	}
}

//AddUser adds a new user to the server
func (server *Server) AddUser(user *UserMeetingParams) {
	server.AddUserCh <- user
	return
}

//RemoveUser removes a new user from the server
func (server *Server) RemoveUser(user *UserMeetingParams) {
	log.Println("Removing user")
	server.RemoveUserCh <- user
}

//ProcessNewIncomingMessage processes incoming message
func (server *Server) ProcessNewIncomingMessage(message Message) {
	fmt.Println("message received from user", message)
	server.NewIncomingMessageCh <- message
}

//Done signals that it is done
func (server *Server) Done() {
	server.DoneCh <- true
}

//SendPastMessages sends all the past messages to a user
func (server *Server) SendPastMessages(user *UserMeetingParams) {
	for _, msg := range server.Messages {
		user.Write(msg)
	}
}

//Err reports error from the server
func (server *Server) Err(err error) {
	server.ErrorCh <- err
}

//SendAll sends the message to all the users present in the chatroom
func (server *Server) SendAll(msg Message) {
	for _, user := range server.ConnectedUsers {
		user.Write(msg)
	}
}

//Listen listens and responds to requests in the chatroom
func (server *Server) Listen() {
	log.Println("chatroom Server Listening .....")
	for {
		select {
		// Adding a new user
		case user := <-server.AddUserCh:
			log.Println("Added a new User to the room", user)
			server.ConnectedUsers[user.Username] = user
			server.SendPastMessages(user)
			server.AddInfoToDB(user, "user "+user.Username+" added")
		//removing a new user
		case user := <-server.RemoveUserCh:
			log.Println("Removing user from chat room")
			delete(server.ConnectedUsers, user.Username)
			server.AddInfoToDB(user, "user "+user.Username+" removed")

		case msg := <-server.NewIncomingMessageCh:
			server.Messages = append(server.Messages, msg)
			server.SendAll(msg)
		case err := <-server.ErrorCh:
			log.Println("Error : ", err)
		case <-server.DoneCh:
			return
			//TODO: clear everything from the server connected to that meeting and shut down the server
		}
	}
}

func (server *Server) handleGetAllMessages(responseWriter http.ResponseWriter, request *http.Request) {
	json.NewEncoder(responseWriter).Encode(server)
}

//AddInfoToDB records in db the action
func (server *Server) AddInfoToDB(user *UserMeetingParams, action string) {
	measurement := strconv.Itoa(int(server.MeetingParams.Name))
	tags := map[string]string{}
	fields := map[string]interface{}{
		"action": action,
	}
	t := time.Now()
	DBwrite(measurement, tags, fields, t)
}
