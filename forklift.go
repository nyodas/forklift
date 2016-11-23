package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	forkliftCmd "github.com/nyodas/forklift/cmd"
	"github.com/nyodas/forklift/logstreamer"

	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
)

type Message struct {
	Type    string
	Content string
	Args    []string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var addr = flag.String("addr", "localhost:8080", "http service address")
var commandName = flag.String("c", "sleep", "Command to run")
var commandCwd = flag.String("cpwd", "/", "Cwd for the command")
var logLevel = flag.String("L", "info", "Loglevel  (default is INFO)")

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logs.WithE(err).WithField("from", r.RemoteAddr).Error("Error with the websocket upgrade")
		return
	}
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	logStreamerOut := logstreamer.NewWsLogstreamer(logger, "stdout", false, *c)
	defer logStreamerOut.Close()
	logStreamerErr := logstreamer.NewWsLogstreamer(logger, "stderr", true, *c)
	defer logStreamerErr.Close()
	forkliftCmd := forkliftCmd.NewForkliftCommand(*commandName, *commandCwd, c, logStreamerOut, logStreamerErr)

	defer c.Close()
	for {
		m := Message{}
		err := c.ReadJSON(&m)
		if err != nil && (websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err)) {
			forkliftCmd.SockClosed = true
			logs.WithE(err).Info("Socket is closed")
			break
		} else if err != nil {
			logs.WithE(err).Error("Error reading json.")
			break
		}
		if m.Type == "command" {
			logs.WithField("cmd", *commandName).Info("Launching command")
			forkliftCmd.Args = m.Args
			forkliftCmd.Prepare()
			go forkliftCmd.Start()
		}
		if m.Type == "kill" {
			logs.WithField("cmd", *commandName).Info("Killing command")
			forkliftCmd.Stop()
		}
	}
}

func main() {
	flag.Parse()
	level, err := logs.ParseLevel(*logLevel)
	if err != nil {
		logs.WithField("value", logLevel).Fatal("Unknown log level")
	}
	logs.SetLevel(level)

	http.HandleFunc("/echo", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
