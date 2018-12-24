package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"personal_website/models"
	"time"

	"github.com/gorilla/schema"
	uuid "github.com/nu7hatch/gouuid"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

//Page contains info to be sent to templates
type Page struct {
	Title  string
	User   models.User
	IsAuth bool
	Data   interface{} //pass any extra data
}

// Declares all the variables that are to be initialized before running the server
var (
	Templates *template.Template
	Store     *sessions.CookieStore
	LiveUsers map[string]models.User
)

//Init helps in initializing different variables and running functions
func Init() {
	models.DBinit()
	LiveUsers = make(map[string]models.User)
	Templates = template.Must(template.ParseGlob("./html/*.gohtml"))
	//related to session
	authKeyOne := securecookie.GenerateRandomKey(64)
	encryptionKeyOne := securecookie.GenerateRandomKey(32)
	Store = sessions.NewCookieStore(authKeyOne, encryptionKeyOne)
	Store.Options = &sessions.Options{
		// MaxAge:   60 * 15, //15 mins max for a cookie
		HttpOnly: true,
	}
}

//AuthRequired redirects the user to "/" page if not logged in
func AuthRequired(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, sessionID := GetSessionDetails(r)
		if username == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if LiveUsers[username.(string)].SessionID != sessionID {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		handler.ServeHTTP(w, r)
	}
}

//RootHandler takes care of the "/" route
func RootHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	username, id := GetSessionDetails(r)
	if username != nil {
		fmt.Println("username in cookie in ROOT", username)
		user = models.GetUser(username.(string), "/")
		if user.Username == "" {
			log.Println("not a valid username and user in RootHandler", user)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if user.SessionID == id {
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
	} else {
		log.Println("no username in request for rootHandler")
	}
	data := Page{Title: "Home Page", User: user, IsAuth: false}
	tErr := Templates.ExecuteTemplate(w, "index", data)
	if tErr != nil {
		log.Println("failed to execute '/' template", tErr)
	}
	return
}

//LogInPostHandler logins a user
func LogInPostHandler(w http.ResponseWriter, r *http.Request) {
	//getting form data
	err := r.ParseForm()
	if err != nil {
		log.Println("failed to parse form during log in")
	}
	var userForm models.User
	decoder := schema.NewDecoder()
	err = decoder.Decode(&userForm, r.PostForm)
	if err != nil {
		log.Println("failed to parse form from client", err)
	}
	//getting user info from DB
	userDB := models.GetUser(userForm.Username, "/login")
	if userDB.Username == "" {
		log.Println("failed to get user from DB log in")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	password := userDB.Password
	if password != userForm.Password {
		log.Println("passwords dont match", password, userForm.Password)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	log.Println("passwords matched")
	//creating new sessiontoken
	sessionToken, uuidErr := uuid.NewV4()
	if uuidErr != nil {
		log.Println("failed to generate a uuid", uuidErr)
	}
	sessionTokenString := sessionToken.String()
	user := models.GetUser(userForm.Username, "/login")
	if user.Username == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	//writing the new session string to the DB
	measurement := "people"
	t, _ := time.Parse(time.RFC3339Nano, user.UTIME)
	tags := map[string]string{
		"username": user.Username,
		"password": user.Password,
	}
	fields := map[string]interface{}{
		"name":       user.Name,
		"session_id": sessionTokenString,
		"admin":      user.IsAdmin,
	}
	models.DBwrite(measurement, tags, fields, t)
	//creating neww session
	session, sErr := Store.Get(r, "session")
	if sErr != nil {
		log.Println("failed to get a session in LogInHandler", sErr)

	}
	session.Values["username"] = user.Username
	session.Values["session_id"] = sessionTokenString
	saveErr := session.Save(r, w)
	if saveErr != nil {
		log.Println("session saving error", saveErr)
	}
	user.SessionID = sessionTokenString
	LiveUsers[user.Username] = user
	http.Redirect(w, r, "/home", http.StatusSeeOther)
	return
}

//GetSessionDetails gets the username and session id from the cookie
func GetSessionDetails(r *http.Request) (username, sessionID interface{}) {
	session, sErr := Store.Get(r, "session")
	if sErr != nil {
		log.Println("failed to get a session in GetSessionDetails", sErr)
	}
	username, _ = session.Values["username"]
	sessionID, _ = session.Values["session_id"]
	return
}

//HomeHandler executes the template after the user logins
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	username, _ := GetSessionDetails(r)
	user := LiveUsers[username.(string)]
	data := Page{Title: "Home Page", User: user, IsAuth: true}
	tErr := Templates.ExecuteTemplate(w, "home", data)
	if tErr != nil {
		log.Println("failed to execute '/home' template", tErr)
	}
	return
}

//TestHandler is used to test random pages and routes
func TestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "this is the start to a another thing")
}

//LogOutGetHandler logs out the user
func LogOutGetHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := Store.Get(r, "session")
	username, _ := session.Values["username"]
	delete(LiveUsers, username.(string))
	session.Values["username"] = nil
	session.Values["session_id"] = nil
	session.Options.MaxAge = -1 //very important
	sErr := session.Save(r, w)
	if sErr != nil {
		log.Println("failed to update session during logout", sErr)
	}
	log.Println(username, "user successfully logged out")
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return
}

//MeetingsHandler lists out all the meetings after querying the database
func MeetingsHandler(w http.ResponseWriter, r *http.Request) {
	//get info of meetings from database
	meetingsDB := models.GetMeetings("/meetings")
	data := Page{Title: "Home Page", IsAuth: true, Data: meetingsDB}
	tErr := Templates.ExecuteTemplate(w, "meetings", data)
	if tErr != nil {
		log.Println("failed to execute '/meetings' template", tErr)
	}
	return
}

//CreateMeetingHandler creates and adds a meeting to the database
func CreateMeetingHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("failed to parse form during create meeting")
	}
	var meeting models.Meeting
	decoder := schema.NewDecoder()
	err = decoder.Decode(&meeting, r.PostForm)
	if err != nil {
		log.Println("failed to parse form from client in createMeeting", err)
	}
	measurement := "meetings"
	tags := map[string]string{}
	fields := map[string]interface{}{
		"name":             float64(meeting.Name),
		"num_attendees":    float64(meeting.NumAttendees),
		"time_space":       float64(meeting.TimeSpace),
		"time_diff":        float64(meeting.TimeDiff),
		"action_time_diff": float64(meeting.ActionTimeDiff),
		"no_cntrl_ents":    float64(meeting.NoCntrlEnts),
	}
	t := time.Now()
	fmt.Println("time now is", t)
	models.DBwrite(measurement, tags, fields, t)
	http.Redirect(w, r, "/meetings", http.StatusSeeOther)
	return
}

func main() {
	Init()

	r := mux.NewRouter()
	r.HandleFunc("/", RootHandler).Methods("GET")
	r.HandleFunc("/login", LogInPostHandler).Methods("POST")
	r.HandleFunc("/logout", AuthRequired(LogOutGetHandler)).Methods("GET")
	r.HandleFunc("/home", AuthRequired(HomeHandler)).Methods("GET")
	r.HandleFunc("/meetings", AuthRequired(MeetingsHandler)).Methods("GET")
	r.HandleFunc("/createMeeting", AuthRequired(CreateMeetingHandler)).Methods("POST")

	r.HandleFunc("/test", TestHandler).Methods("GET")

	r.PathPrefix("/public/").Handler(http.FileServer(http.Dir(".")))

	fmt.Println("running server on port 9090")
	http.ListenAndServe(":9090", r)
}
