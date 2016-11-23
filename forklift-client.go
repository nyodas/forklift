package main

import (
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"os/signal"

	"strings"

	"github.com/gorilla/websocket"
	"github.com/n0rad/go-erlog"
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/nyodas/forklift/erlog-forklift"
)

type Message struct {
	Type    string
	Content string
	Args    []string
}

type commandOutputLog struct {
	Type   string
	Msg    string
	Prefix string
}

var addr = flag.String("addr", "localhost:8080", "http service address")
var args = flag.String("args", "10", "Args.")
var logLevel = flag.String("L", "info", "Loglevel  (default is INFO)")

func sendMessage(msg Message, c *websocket.Conn) {
	messageJson, err := json.Marshal(msg)
	if err != nil {
		logs.WithE(err).WithField("args", msg.Args).WithField("type", msg.Type).WithField("content", msg.Content).Error("Failed to Marshal Json")
		return
	}
	err = c.WriteMessage(websocket.TextMessage, messageJson)
	if err != nil {
		logs.WithE(err).WithField("args", msg.Args).WithField("type", msg.Type).WithField("content", msg.Content).Error("Failed to Send message")
		return
	}
}

func main() {
	flag.Parse()
	customAppender := erlog_forklift.NewForkliftErlogWriterAppender(os.Stdout)
	logWs := logs.GetLog("logWs")
	logWs.(*erlog.ErlogLogger).Appenders = []erlog.Appender{customAppender}

	level, err := logs.ParseLevel(*logLevel)
	if err != nil {
		logs.WithField("value", logLevel).Fatal("Unknown log level")
	}
	logs.SetLevel(level)
	logs.WithField("addr", *addr).WithField("args", *args).Debug("Arguments")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	logs.WithField("url", u.String()).Info("Connecting")

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logs.WithE(err).WithField("url", u.String()).Fatal("Failed to connect.")
	}

	isClosed := false
	done := make(chan struct{})
	go func() {
		defer c.Close()
		for {
			m := commandOutputLog{}
			err := c.ReadJSON(&m)
			if err != nil {
				logs.WithE(err).Info("Socket is closed.")
				close(done)
				isClosed = true
				break
			}
			m.Msg = strings.TrimRight(m.Msg, "\n")
			if m.Prefix == "stdout" {
				logWs.Info(m.Msg)
			} else if m.Prefix == "stderr" {
				logWs.Error(m.Msg)
			}
		}
	}()

	logs.Info("Sending command")
	msg := Message{
		Type:    "command",
		Content: "ls",
		Args:    strings.Split(*args, " "),
	}
	sendMessage(msg, c)

	for {
		select {
		case <-interrupt:
			logs.Info("Received interrupt... Sending kill")
			messageKill := Message{
				Type: "kill",
			}
			sendMessage(messageKill, c)
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logs.WithE(err).Error("Failed to Close the websocket connection.")
				return
			}
		case <-done:
			if isClosed == false {
				logs.Info("Closing Websocket")
				c.Close()
			}
			logs.Debug("We're done.Exiting")
			os.Exit(0)
			return
		}
	}
}
