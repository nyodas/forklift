package logstreamer

type LogStreamer interface {
	Write([]byte) (int, error)
	Close() error
	Flush() error
	OutputLines() error
	FlushRecord() string
}
