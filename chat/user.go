package chat

import (
	"encoding/json"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
	"github.com/gorilla/websocket"
)

type User struct {
	SessionID string
	Data
}

type Data struct {
	UID  uint
	Conn *websocket.Conn
	Anon bool
}

type ProcessWSMessage struct {
	From *User
	WSM  interface{}
}

//easyjson:json
type ReceivedWSMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

func (u *User) Listen() {
	for {
		m := &ReceivedWSMessage{}
		_, raw, err := u.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				if !u.Anon {
					logger.Infof("User %v with session %v was disconnected", u.UID, u.SessionID)
				} else {
					logger.Infof("anonymous user with temp session %v was disconnected", u.SessionID)
				}
			} else {
				logger.Error(err)
			}
			chat.Leave <- u
			return
		}
		err = m.UnmarshalJSON(raw)
		if err != nil {
			logger.Error(err)
			continue
		}

		logger.Infof("Read WSMessage: %v", *m)

		chat.Send <- &ProcessWSMessage{
			From: u,
			WSM:  m,
		}
	}
}
