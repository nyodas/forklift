package logstreamer

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/wsxiaoys/terminal"
)

type LogStreamerTerm struct {
	buf *bytes.Buffer
	// if true, saves output in memory
	record  bool
	persist string
	prefix  string
	name    string
}

func NewLogStreamerTerm(prefix string, record bool, name string) *LogStreamerTerm {
	streamer := &LogStreamerTerm{
		buf:     bytes.NewBuffer([]byte("")),
		prefix:  prefix,
		record:  record,
		persist: "",
		name:    name,
	}

	return streamer
}

func (l *LogStreamerTerm) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.OutputLines()
	return
}

func (l *LogStreamerTerm) Close() error {
	if err := l.Flush(); err != nil {
		return err
	}
	l.buf = bytes.NewBuffer([]byte(""))
	return nil
}

func (l *LogStreamerTerm) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

func (l *LogStreamerTerm) OutputLines() error {
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

func (l *LogStreamerTerm) FlushRecord() string {
	buffer := l.persist
	l.persist = ""
	return buffer
}

func (l *LogStreamerTerm) out(str string) {
	if len(str) < 1 {
		return
	}
	if l.record == true {
		l.persist = l.persist + str
	}
	color := "g"
	if l.prefix == "stderr" {
		color = "r"
	}
	terminal.Stdout.
		Color(color).Print(fmt.Sprintf("[%s][%s] ", l.prefix, l.name)).
		Reset().Print(str)
}
