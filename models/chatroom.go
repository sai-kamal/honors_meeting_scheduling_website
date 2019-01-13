package models

import (
	"encoding/json"
	"log"
	"net/http"
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
	Messages             []*Message
	AddUserCh            chan *UserMeetingParams
	RemoveUserCh         chan *UserMeetingParams
	NewIncomingMessageCh chan *Message
	ErrorCh              chan error
	DoneCh               chan bool
	MeetingParams        Meeting //contains info about the meeting
}

//NewServer creates a new server for the chatroom
func NewServer(meeting Meeting) *Server {
	ConnectedUsers := make(map[string]*UserMeetingParams)
	Messages := []*Message{}
	AddUserCh := make(chan *UserMeetingParams)
	RemoveUserCh := make(chan *UserMeetingParams)
	NewIncomingMessage := make(chan *Message)
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
	log.Println("Adding user")
	server.AddUserCh <- user
}

//RemoveUser removes a new user from the server
func (server *Server) RemoveUser(user *UserMeetingParams) {
	log.Println("Removing user")
	server.RemoveUserCh <- user
}

//ProcessNewIncomingMessage processes incoming message
func (server *Server) ProcessNewIncomingMessage(message *Message) {
	server.NewIncomingMessageCh <- message
}

//Done signals that it is done
func (server *Server) Done() {
	server.DoneCh <- true
}

//SendPastMessages sends all the past messages to a user
func (server *Server) SendPastMessages(user *UserMeetingParams) {
	for _, msg := range server.Messages {
		//  log.Println("In sendPastMessages writing ",msg)
		user.Write(msg)
	}
}

//Err reports error from the server
func (server *Server) Err(err error) {
	server.ErrorCh <- err
}

//SendAll sends the message to all the users present in the chatroom
func (server *Server) SendAll(msg *Message) {
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
			log.Println("Added a new UserMeetingParams to the room")
			server.ConnectedUsers[user.Username] = user
			log.Println("Now ", len(server.ConnectedUsers), " users are connected to chat room")
			server.SendPastMessages(user)
		case user := <-server.RemoveUserCh:
			log.Println("Removing user from chat room")
			delete(server.ConnectedUsers, user.Username)
		case msg := <-server.NewIncomingMessageCh:
			server.Messages = append(server.Messages, msg)
			//TODO: update database with all the messages that have been used
			server.SendAll(msg)
		case err := <-server.ErrorCh:
			log.Println("Error : ", err)
		case <-server.DoneCh:
			return
			//TODO: clear everything from the server connected to that meeting
		}
	}
}

func (server *Server) handleGetAllMessages(responseWriter http.ResponseWriter, request *http.Request) {
	json.NewEncoder(responseWriter).Encode(server)
}
