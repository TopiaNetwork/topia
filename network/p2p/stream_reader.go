package p2p

import (
	"io"
	"time"
)

type ReaderDeadline interface {
	Read([]byte) (int, error)
	SetReadDeadline(time.Time) error
}

type StreamReader struct {
	rd ReaderDeadline

	waitPerByte time.Duration
	wait        time.Duration
	maxWait     time.Duration
}

// New creates an Incremental Reader Timeout, with minimum sustained speed of
// minSpeed bytes per second and with maximum wait of maxWait
func NewStreamReader(rd ReaderDeadline, minSpeed int64, maxWait time.Duration) io.Reader {
	return &StreamReader{
		rd:          rd,
		waitPerByte: time.Second / time.Duration(minSpeed),
		wait:        maxWait,
		maxWait:     maxWait,
	}
}

type errNoWait struct{}

func (err errNoWait) Error() string {
	return "wait time exceeded"
}
func (err errNoWait) Timeout() bool {
	return true
}

func (sr *StreamReader) Read(buf []byte) (int, error) {
	start := time.Now()
	if sr.wait == 0 {
		return 0, errNoWait{}
	}

	err := sr.rd.SetReadDeadline(start.Add(sr.wait))
	if err != nil {

	}

	n, err := sr.rd.Read(buf)

	_ = sr.rd.SetReadDeadline(time.Time{})
	if err == nil {
		dur := time.Since(start)
		sr.wait -= dur
		sr.wait += time.Duration(n) * sr.waitPerByte
		if sr.wait < 0 {
			sr.wait = 0
		}
		if sr.wait > sr.maxWait {
			sr.wait = sr.maxWait
		}
	}
	return n, err
}
