package main

import (
	"flag"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/middleware"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/session"

	"chat/chat"
)

func main() {
	dbConnStr := flag.String("db_connstr", "postgres@localhost:5432", "postgresql connection string")
	dbName := flag.String("db_name", "postgres", "database name")
	authConnStr := flag.String("auth_connstr", "localhost:8081", "auth-service connection string")
	flag.Parse()

	l := logger.InitLogger()
	defer func() {
		err := l.Sync()
		if err != nil {
			logger.Errorf("error while syncing log data: %v", err)
		}
	}()

	dm := database.InitDatabaseManager(*dbConnStr, *dbName)
	defer dm.Close()

	sm := session.ConnectSessionManager(*authConnStr)
	defer sm.Close()

	chat := chat.InitChat(dm)
	go chat.Run()

	http.HandleFunc("/chat/ws", middleware.RecoverMiddleware(middleware.AccessLogMiddleware(
		middleware.CORSMiddleware(middleware.SessionMiddleware(http.HandlerFunc(ConnectChat), sm)))))

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
