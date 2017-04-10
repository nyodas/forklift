package logstreamer

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/nyodas/forklift/msg"
)

type LogStreamerWs struct {
	LoggerStdout *LogStreamerTerm
	LoggerStderr *LogStreamerTerm
	buf          *bytes.Buffer
	// if true, saves output in memory
	record  bool
	persist string
	prefix  string
	ws      websocket.Conn
}

func NewLogStreamerWs(prefix string, record bool, wsConn websocket.Conn, name string) *LogStreamerWs {
	streamer := &LogStreamerWs{
		LoggerStdout: NewLogStreamerTerm("stdout", false, name),
		LoggerStderr: NewLogStreamerTerm("stderr", false, name),
		buf:          bytes.NewBuffer([]byte("")),
		prefix:       prefix,
		record:       record,
		ws:           wsConn,
		persist:      "",
	}

	return streamer
}

func (l *LogStreamerWs) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.OutputLines()
	return
}

func (l *LogStreamerWs) Close() error {
	if err := l.Flush(); err != nil {
		return err
	}
	l.buf = bytes.NewBuffer([]byte(""))
	return nil
}

func (l *LogStreamerWs) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

func (l *LogStreamerWs) OutputLines() error {
	for {
		line, err := l.buf.ReadString('\n')

		if len(line) > 0 {
			if strings.HasSuffix(line, "\n") {
				l.out(line)
			} else {
				// put back into buffer, it's not a complete line yet
				//  Close() or Flush() have to be used to flush out
				//  the last remaining line if it does not end with a newline
				if _, err := l.buf.WriteString(line); err != nil {
					return err
				}
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (l *LogStreamerWs) FlushRecord() string {
	buffer := l.persist
	l.persist = ""
	return buffer
}

func (l *LogStreamerWs) out(str string) {
	if len(str) < 1 {
		return
	}
	logMsg := msg.CommandOutputLog{
		Message: msg.Message{
			Type:    "log",
			Content: str,
		},
		Prefix: l.prefix,
	}
	if l.record == true {
		l.persist = l.persist + str
	}

	if err := logMsg.Send(&l.ws); err != nil {
		fmt.Println(err)
	}
	if l.prefix == "stdout" {
		l.LoggerStdout.out(str)
		l.LoggerStdout.Flush()
	} else {
		l.LoggerStderr.out(str)
		l.LoggerStderr.Flush()
	}
}
