package msg

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/n0rad/go-erlog/logs"
)

type MessageSender interface {
	Send(c *websocket.Conn) (err error)
}

type Message struct {
	Type    string
	Content string
}

type CommandRequest struct {
	Message
	Args []string
}

type CommandOutputLog struct {
	Message
	Prefix string
}

func Send(c *websocket.Conn, msg interface{}) (err error) {
	messageJson, err := json.Marshal(msg)
	if err != nil {
		logs.WithE(err).WithField("msg", msg).
			Error("Failed to Marshal Json")
	}
	err = c.WriteMessage(websocket.TextMessage, messageJson)
	if err != nil {
		logs.WithE(err).WithField("msg", msg).
			Error("Failed to Send message")
	}
	return err

}
func (msg *Message) Send(c *websocket.Conn) (err error) {
	return Send(c, msg)
}

func (msg *CommandRequest) Send(c *websocket.Conn) (err error) {
	return Send(c, msg)
}
func (msg *CommandOutputLog) Send(c *websocket.Conn) (err error) {
	return Send(c, msg)
}
