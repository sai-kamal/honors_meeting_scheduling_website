package models

import (
	"fmt"
	"log"
)

//User struct defines the necessary parameters related to the user
type User struct {
	Username  string `json:"username" schema:"username"`
	Password  string `json:"password" schema:"password"`
	IsAuth    bool   `schema:"-"`
	Name      string `json:"name" schema:"name"`
	IsAdmin   bool   `json:"admin" schema:"admin"`
	SessionID string `json:"session_id" schema:"session_id"`
	UTIME     string `json:"time" schema:"time"`
}

//GetUser gets the user info from the database
func GetUser(username, route string) User {
	var user User
	queryStr := fmt.Sprintf("SELECT * FROM people WHERE \"username\"='%v'", username)
	resp, qErr := DBquery(queryStr)
	if qErr != nil || len(resp[0].Series) == 0 {
		log.Println("failed to get user data from DB in ", route, qErr)
		return user
	}
	userArr := resp[0].Series[0].Values[0]
	user.UTIME, user.IsAdmin, user.Name, user.Password, user.SessionID, user.Username = userArr[0].(string), userArr[1].(bool), userArr[2].(string), userArr[3].(string), userArr[4].(string), userArr[5].(string)
	return user
}
