package logstreamer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/gorilla/websocket"
)

type LogstreamerWs struct {
	Logger *log.Logger
	buf    *bytes.Buffer
	// if true, saves output in memory
	record  bool
	persist string
	prefix  string
	ws      websocket.Conn
}

type commandOutputLog struct {
	Type   string
	Msg    string
	Prefix string
}

func NewWsLogstreamer(logger *log.Logger, prefix string, record bool, wsConn websocket.Conn) *LogstreamerWs {
	streamer := &LogstreamerWs{
		Logger:  logger,
		buf:     bytes.NewBuffer([]byte("")),
		prefix:  prefix,
		record:  record,
		ws:      wsConn,
		persist: "",
	}

	return streamer
}

func (l *LogstreamerWs) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.OutputLines()
	return
}

func (l *LogstreamerWs) Close() error {
	if err := l.Flush(); err != nil {
		return err
	}
	l.buf = bytes.NewBuffer([]byte(""))
	return nil
}

func (l *LogstreamerWs) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

func (l *LogstreamerWs) OutputLines() error {
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

func (l *LogstreamerWs) FlushRecord() string {
	buffer := l.persist
	l.persist = ""
	return buffer
}

func (l *LogstreamerWs) out(str string) {
	if len(str) < 1 {
		return
	}
	msg := commandOutputLog{
		Type:   "log",
		Msg:    str,
		Prefix: l.prefix,
	}
	msgJson, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	if l.record == true {
		l.persist = l.persist + str
	}
	if err := l.ws.WriteMessage(1, msgJson); err != nil {
		fmt.Println(err)
	}
	l.Logger.Print(str)
}
