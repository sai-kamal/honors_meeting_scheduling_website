package models

import (
	"fmt"
	"log"

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
	return
}

//Write sends the user a message
func (user *UserMeetingParams) Write(message Message) {
	select {
	case user.OutgoingMessageCh <- &message:
	default:
		user.Server.RemoveUser(user)
		err := fmt.Errorf("userMeetingParams %s is disconnected", user.Username)
		user.Server.Err(err)
	}
}

//Done signals the user that it is done
func (user *UserMeetingParams) Done() {
	user.DoneCh <- true
}

//Listen writes and reads for the user
func (user *UserMeetingParams) Listen() {
	go user.listenWrite()
	user.listenRead()
}

func (user *UserMeetingParams) listenWrite() {
	for {
		select {
		//send message to user
		case msg := <-user.OutgoingMessageCh:
			log.Println("message is being sent from server to user", user, msg)
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
				user.Server.Err(err)
			} else {
				user.Server.ProcessNewIncomingMessage(messageObj)
			}
		}
	}
}
