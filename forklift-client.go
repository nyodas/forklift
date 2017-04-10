package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/mgutz/str"
	"github.com/n0rad/go-erlog"
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/nyodas/forklift/erlog-forklift"
	"github.com/nyodas/forklift/msg"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var args = flag.String("args", "-l -a -h 'yolo'", "Args.")
var execCmd = flag.String("e", "consume", "shortname of the command")
var remoteArgs = flag.Bool("remoteargs", false, "print remote args")
var logLevel = flag.String("L", "info", "Loglevel  (default is INFO)")

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
			m := msg.CommandOutputLog{}
			err := c.ReadJSON(&m)
			if err != nil {
				logs.WithE(err).Info("Socket is closed.")
				close(done)
				isClosed = true
				break
			}
			if m.Type == "log" {
				m.Content = strings.TrimRight(m.Content, "\n")
				if m.Prefix == "stdout" {
					logWs.Info(m.Content)
				} else if m.Prefix == "stderr" {
					logWs.Error(m.Content)
				}
			}
			if m.Type == "args" {
				fmt.Printf(m.Content + "\n")
			}
		}
	}()

	msgRequest := msg.CommandRequest{
		Message: msg.Message{
			Type:    "exec",
			Content: *execCmd,
		},
		Args: str.ToArgv(*args),
	}

	logs.Info("Sending command")
	if *remoteArgs {
		logs.Info("Gettings current args")
		msgRequest.Type = "args"
	}
	_ = msgRequest.Send(c)
	for {
		select {
		case <-interrupt:
			logs.Info("Received interrupt... Sending kill")
			messageKill := msg.Message{
				Type: "kill",
			}
			_ = messageKill.Send(c)
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				logs.WithE(err).Error("Failed to Close the websocket connection.")
				return
			}
		case <-done:
			if isClosed == false {
				logs.Info("Closing Websocket")
				if c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")) != nil {
					c.Close()
				}
			}
			logs.Debug("We're done.Exiting")
			os.Exit(0)
			return
		}
	}
}
