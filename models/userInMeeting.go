package models

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

const channelBufSize = 100

//UserMeetingParams contains info about users who joined a meeting
type UserMeetingParams struct {
	MeetingName      int64  `json:"meeting_name" schema:"meeting_name"`
	Username         string `json:"username" schema:"username"`
	AgentExpectation int64  `json:"delay" schema:"delay"`
	// CntrlEnt         int64  `json:"cntrl_ent" schema:"cntrl_ent"`
	Importance int64 `json:"importance" schema:"importance"` // to represent the meeting importance to the user
	//TODO: resp = 1
	Conn              *websocket.Conn
	Server            *Server
	OutgoingMessageCh chan *Message
	DoneCh            chan bool
}

// CreateUserMeetingParams creates a new UserMeetingParams
func CreateUserMeetingParams(conn *websocket.Conn, server *Server, user *UserMeetingParams) {
	if conn == nil {
		log.Println("connection cannot be nil", user)
	}
	if server == nil {
		log.Println(" Server cannot be nil", user)
	}
	user.Conn = conn
	user.Server = server
	user.OutgoingMessageCh = make(chan *Message, channelBufSize)
	user.DoneCh = make(chan bool)
	return
}

//GetConn returns conn of the user
func (user *UserMeetingParams) GetConn() *websocket.Conn {
	return user.Conn
}

//GetServer returns server of the user
func (user *UserMeetingParams) GetServer() *Server {
	return user.Server
}

//Write sends the user a message
func (user *UserMeetingParams) Write(message *Message) {
	select {
	case user.OutgoingMessageCh <- message:
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
	log.Println("Listening to write to client")
	for {
		select {
		//send message to user
		case msg := <-user.OutgoingMessageCh:
			//  log.Println("send in listenWrite for user :",user.id, msg)
			user.Conn.WriteJSON(&msg)
			// receive done request
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
				user.Server.ProcessNewIncomingMessage(&messageObj)
			}
		}
	}
}
