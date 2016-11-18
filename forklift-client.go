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
var args = flag.String("args", "-ls .", "Args.")
var logLevel = flag.String("L", "info", "Loglevel  (default is INFO)")

func main() {
	flag.Parse()
	erlogFactory := erlog.NewErlogFactory()
	logs.RegisterLoggerFactory(erlogFactory)
	customAppender := erlog_forklift.NewForkliftErlogWriterAppender(os.Stdout)
	logWs := erlogFactory.GetCustomLog("logws", customAppender)

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
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer c.Close()
		for {
			m := commandOutputLog{}
			err := c.ReadJSON(&m)
			if err != nil {
				logs.WithE(err).Error("Failed to read the incoming message.")
				close(done)
				return
			}
			m.Msg = strings.TrimRight(m.Msg, "\n")
			if m.Prefix == "stdout" {
				logWs.Info(m.Msg)
			} else if m.Prefix == "stderr" {
				logWs.Error(m.Msg)
			}

			if _, _, err := c.NextReader(); err != nil {
				close(done)
				return
			}
		}
	}()

	message := Message{
		Type:    "command",
		Content: "ls",
	}
	message.Args = strings.Split(*args, " ")
	messageJson, err := json.Marshal(message)
	if err != nil {
		logs.WithE(err).WithField("args", message.Args).WithField("type", message.Type).WithField("content", message.Content).Error("Failed to Marshal Json")
		return
	}
	err = c.WriteMessage(websocket.TextMessage, messageJson)
	if err != nil {
		logs.WithE(err).WithField("args", message.Args).WithField("type", message.Type).WithField("content", message.Content).Error("Failed to Send message")
		return
	}

	for {
		select {
		case <-interrupt:
			logs.Info("Received interrupt... Closing")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logs.WithE(err).Error("Failed to Close the websocket connection.")
				return
			}
		case <-done:
			logs.Info("Closing")
			c.Close()
			os.Exit(0)
			return
		}
	}
}
