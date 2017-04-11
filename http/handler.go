package http

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/n0rad/go-erlog/logs"
	"github.com/nyodas/forklift/forkliftcmd"
	"github.com/nyodas/forklift/logstreamer"
	"github.com/nyodas/forklift/msg"
	"github.com/nyodas/forklift/runner"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Handler struct {
}

func (h *Handler) ExecRemoteCmd(w http.ResponseWriter, r *http.Request) {
	var forkliftExec *runner.Runner
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
		cmdName := m.Content
		if m.Type == "exec" || m.Type == "command" {
			cmdName := m.Content
			configLocalCmd := forkliftConfig.FindRemoteCommand(cmdName)
			logs.WithField("command", configLocalCmd.Shortname).
				WithField("args", m.Args).
				Info("Launching command")
			logStreamerOut := logstreamer.NewLogStreamerWs("stdout", true, *c, configLocalCmd.Path)
			logStreamerErr := logstreamer.NewLogStreamerWs("stderr", true, *c, configLocalCmd.Path)
			forkliftExec = runner.NewRunner(configLocalCmd.Path, configLocalCmd.Cwd, []string{""})
			forkliftExec.Args = m.Args
			forkliftExec.Prepare()
			forkliftExec.SetLogger(logStreamerOut, logStreamerErr)
			go func() {
				forkliftExec.Start()
				logStreamerOut.Close()
				logStreamerErr.Close()
				h.closeWS(c)
			}()
		}
		if m.Type == "args" {
			configRemoteCmd := forkliftConfig.FindLocalCommand(cmdName)
			logs.WithField("args", configRemoteCmd.Args).Debug("Gettings Args")
			argsMsg := msg.Message{Type: "args", Content: configRemoteCmd.Args}
			_ = argsMsg.Send(c)
			h.closeWS(c)
		}
		if m.Type == "kill" {
			logs.WithField("command", cmdName).Info("Killing command")
			forkliftExec.Stop()
		}
	}
}

func (h *Handler) closeWS(c *websocket.Conn) {
	if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		logs.WithE(err).Error("Error closing websocket")
		_ = c.Close()
	}
}
