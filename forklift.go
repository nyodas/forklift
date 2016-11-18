package main

import (
	"flag"
	"log"
	"net/http"
	"os/exec"

	"os"

	"github.com/gorilla/websocket"
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
var commandName = flag.String("c", "ls", "Command to run")
var commandCwd = flag.String("cpwd", "/", "Cwd for the command")
var logLevel = flag.String("L", "info", "Loglevel  (default is INFO)")

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logs.WithE(err).WithField("from", r.RemoteAddr).Error("Error with the websocket upgrade")
		return
	}
	defer c.Close()
	for {
		m := Message{}
		if err := c.ReadJSON(&m); err != nil {
			logs.WithE(err).Error("Error reading json.")
			break
		}
		if m.Type == "command" {
			launch(*commandName, m.Args, c)
		}
	}
}

func launch(commandName string, argsCommand []string, conn *websocket.Conn) {
	// Create a logger (your app probably already has one)
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	// Setup a streamer that we'll pipe cmd.Stdout to
	logStreamerOut := logstreamer.NewWsLogstreamer(logger, "stdout", false, *conn)
	defer logStreamerOut.Close()
	// Setup a streamer that we'll pipe cmd.Stderr to.
	// We want to record/buffer anything that's written to this (3rd argument true)
	logStreamerErr := logstreamer.NewWsLogstreamer(logger, "stderr", true, *conn)
	defer logStreamerErr.Close()

	// Execute something that succeeds
	cmd := exec.Command(
		commandName,
		argsCommand...,
	)
	cmd.Stderr = logStreamerErr
	cmd.Stdout = logStreamerOut
	cmd.Dir = *commandCwd

	// Reset any error we recorded
	logStreamerErr.FlushRecord()
	logs.WithField("command", commandName).WithField("args", argsCommand).Debug("Executing command.")
	// Execute command
	if err := cmd.Start(); err != nil {
		logs.WithE(err).WithField("command", commandName).WithField("args", argsCommand).Error("Error executing command")

	}

	if err := cmd.Wait(); err != nil {
		logs.WithE(err).WithField("command", commandName).WithField("args", argsCommand).Error("Error executing command")
	}
	logStreamerErr.Flush()

	if err := conn.Close(); err != nil {
		logs.WithE(err).Error("Error closing websocket")
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
