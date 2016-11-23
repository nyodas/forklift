package cmd

import (
	"github.com/gorilla/websocket"
	"github.com/n0rad/go-erlog/logs"
	"github.com/nyodas/forklift/logstreamer"
	"os/exec"
)

type ForkliftCommand struct {
	commandName string
	commandCwd  string
	Args        []string
	wsConn      *websocket.Conn
	SockClosed  bool
	logStdout   *logstreamer.LogstreamerWs
	logStderr   *logstreamer.LogstreamerWs
	process     *exec.Cmd
}

func NewForkliftCommand(name string, commandCwd string, c *websocket.Conn, stdoutLogger *logstreamer.LogstreamerWs, stderrLogger *logstreamer.LogstreamerWs) *ForkliftCommand {
	cmd := &ForkliftCommand{
		commandName: name,
		commandCwd:  commandCwd,
		wsConn:      c,
		logStdout:   stdoutLogger,
		logStderr:   stderrLogger,
	}
	return cmd
}

func (r *ForkliftCommand) Prepare() {
	cmd := exec.Command(
		r.commandName,
		r.Args...,
	)
	cmd.Stdout = r.logStdout
	cmd.Stderr = r.logStderr
	cmd.Dir = r.commandCwd
	r.process = cmd
}

func (r *ForkliftCommand) Start() {
	logs.WithField("command", r.commandName).Info("Starting command")
	if err := r.process.Start(); err != nil {
		logs.WithE(err).WithField("command", r.commandName).WithField("args", r.Args).Error("Error executing command")
	}
	if err := r.process.Wait(); err != nil {
		logs.WithE(err).WithField("command", r.commandName).WithField("args", r.Args).Error("Error executing command")
	}
	r.logStderr.Flush()
	r.logStdout.Flush()
	r.SockClosed = true
	if err := r.wsConn.Close(); err != nil && r.SockClosed != false {
		logs.WithE(err).Error("Error closing websocket")
	}
}

func (r *ForkliftCommand) Stop() {
	logs.WithField("command", r.commandName).Info("Stoping command")
	r.process.Process.Kill()
	r.logStderr.Flush()
	r.logStdout.Flush()
}
