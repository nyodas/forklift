package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
	"github.com/juju/errors"
	"github.com/mgutz/str"
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/nyodas/forklift/cmd"
	"github.com/nyodas/forklift/logstreamer"
	"github.com/nyodas/forklift/msg"
	forkliftRunner "github.com/nyodas/forklift/runner"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var addr = flag.String("addr", "localhost:8080", "http service address")
var commandName = flag.String("c", "/bin/ls", "Command to run")
var commandCwd = flag.String("cpwd", "/", "Cwd for the command")
var logLevel = flag.String("L", "info", "Loglevel  (default is INFO)")
var execProc = flag.Bool("e", false, "Exec background process")
var configPath = flag.String("config", "", "Config file path")

func execRemoteCmd(w http.ResponseWriter, r *http.Request) {
	var forkliftExec *forkliftRunner.Runner
	forkliftConfig := forkliftcmd.NewForkliftCommandConfig()
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logs.WithE(err).WithField("from", r.RemoteAddr).
			Error("Error with the websocket upgrade")
		return
	}

	defer c.Close()
	for {
		m := msg.CommandRequest{}
		err := c.ReadJSON(&m)
		if err != nil && (websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err)) {
			logs.WithE(err).Info("Socket closed")
			break
		} else if err != nil {
			logs.WithE(err).Error("Error reading json.")
			break
		}
		if m.Type == "exec" || m.Type == "command" {
			cmdName := m.Content
			configLocalCmd := forkliftConfig.FindRemoteCommand(cmdName)
			logs.WithField("cmd", configLocalCmd.Shortname).
				WithField("args", m.Args).
				Info("Launching command")
			logStreamerOut, logStreamerErr := wsCmdLogger(c, configLocalCmd.Path)
			forkliftExec = forkliftRunner.NewRunner(configLocalCmd.Path, configLocalCmd.Cwd, []string{""})
			forkliftExec.Args = m.Args
			forkliftExec.Prepare()
			forkliftExec.SetLogger(logStreamerOut, logStreamerErr)
			go func() {
				forkliftExec.Start()
				logStreamerOut.Close()
				logStreamerErr.Close()
				closeWS(c)
			}()
		}
		if m.Type == "args" {
			cmdName := m.Content
			configRemoteCmd := forkliftConfig.FindLocalCommand(cmdName)
			logs.WithField("args", configRemoteCmd.Args).Debug("Gettings Args")
			argsMsg := msg.Message{Type: "args", Content: configRemoteCmd.Args}
			_ = argsMsg.Send(c)
			closeWS(c)
		}
		if m.Type == "kill" {
			logs.WithField("cmd", *commandName).Info("Killing command")
			forkliftExec.Stop()
		}
	}
}

func exitHandler(done chan struct{}, runner *forkliftRunner.Runner) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	for {
		select {
		case <-interrupt:
			runner.Stop()
			os.Exit(1)
		case <-done:
			logs.WithField("exitcode", runner.Status).
				Debug("We're done.Exiting")
			os.Exit(runner.Status)
			return
		}
	}
}

func wsCmdLogger(c *websocket.Conn, cmdName string) (logStreamerOut logstreamer.LogStreamer, logStreamerErr logstreamer.LogStreamer) {
	logStreamerOut = logstreamer.NewLogStreamerWs("stdout", false, *c, cmdName)
	logStreamerErr = logstreamer.NewLogStreamerWs("stderr", true, *c, cmdName)
	return logStreamerOut, logStreamerErr
}

func closeWS(c *websocket.Conn) {
	if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		logs.WithE(err).Error("Error closing websocket")
		_ = c.Close()
	}
}

func main() {
	flag.Parse()
	level, err := logs.ParseLevel(*logLevel)
	if err != nil {
		logs.WithField("value", logLevel).Fatal("Unknown log level")
	}
	logs.SetLevel(level)
	file, err := loadConfig(*configPath)
	if err != nil {
		logs.WithE(err).WithField("configfile", configPath).
			Error("Config file empty or missing")
	}
	var cmdConfig *forkliftcmd.ForkliftCommandConfig
	if file != nil {
		if cmdConfig, err = forkliftcmd.MapConfigFile(file); err != nil {
			logs.WithE(err).WithField("configfile", configPath).
				WithField("config", cmdConfig).
				Fatal("Failed to map command config file")
		}
		_ = cmdConfig.SetDefaultCommand(*commandName, *commandCwd)
	}

	if *execProc && file != nil {
		for _, v := range cmdConfig.LocalConfig {
			runner := forkliftRunner.NewRunner(v.Path, v.Cwd, str.ToArgv(v.Args))
			runner.Timeout = v.Timeout
			runner.Oneshot = v.Oneshot
			done := make(chan struct{})
			go func() {
				runner.ExecLoop()
				close(done)
			}()
			go exitHandler(done, runner)
		}
	}
	http.HandleFunc("/echo", execRemoteCmd)
	http.HandleFunc("/exec", execRemoteCmd)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func loadConfig(configPath string) (file []byte, err error) {
	if configPath == "" {
		return nil, errors.New("No config file defined")
	}
	logs.WithField("configfile", configPath).Debug("Loading config")
	file, err = ioutil.ReadFile(configPath)
	return file, err
}
