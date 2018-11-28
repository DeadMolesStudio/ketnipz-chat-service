package main

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/middleware"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/session"

	"chat/chat"
)

func main() {
	l := logger.InitLogger()
	defer l.Sync()

	db := database.InitDB("postgres@chat-db:5432", "chat")
	defer db.Close()

	sm := session.ConnectSessionManager()
	defer sm.Close()

	chat := chat.InitChat()
	go chat.Run()

	http.HandleFunc("/chat/ws", middleware.RecoverMiddleware(middleware.AccessLogMiddleware(
		middleware.CORSMiddleware(middleware.SessionMiddleware(ConnectChat, sm)))))

	logger.Info("starting server at: ", 8083)
	logger.Panic(http.ListenAndServe(":8083", nil))
}

func ConnectChat(w http.ResponseWriter, r *http.Request) {
	u := &chat.User{}
	if r.Context().Value(middleware.KeyIsAuthenticated).(bool) {
		u.SessionID = r.Context().Value(middleware.KeySessionID).(string)
		u.UID = r.Context().Value(middleware.KeyUserID).(uint)
	} else {
		u.Anon = true
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Cannot upgrade connection: ", err)
		return
	}
	u.Conn = conn

	chat.JoinUser(u)
}
