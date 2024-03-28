package cfspeed

import (
	"io"
	"time"
)

type IOEvent struct {
	Timestamp time.Time
	Mode      string
	Size      int
}

type IOSampler struct {
	SizeRead    int64
	SizeWritten int64
	Events      []*IOEvent
}

type SamplingReaderWriter struct {
	IOSampler
	Quota    int64
	GoodThru time.Time
}

func (r *SamplingReaderWriter) Read(p []byte) (int, error) {
	var err error = nil

	size := len(p)
	if r.SizeRead+int64(size) > r.Quota {
		size = int(r.Quota - r.SizeRead)
		err = io.EOF
	}
	if time.Since(r.GoodThru) > 0 {
		size = 0
		err = io.EOF
	}

	r.Events = append(r.Events, &IOEvent{
		Timestamp: time.Now(),
		Mode:      "read",
		Size:      size,
	})
	r.SizeRead += int64(size)

	return size, err
}

func (w *SamplingReaderWriter) Write(p []byte) (int, error) {
	size := len(p)

	w.Events = append(w.Events, &IOEvent{
		Timestamp: time.Now(),
		Mode:      "write",
		Size:      size,
	})
	w.SizeWritten = int64(size)

	var err error = nil
	if w.SizeWritten > w.Quota || time.Since(w.GoodThru) > 0 {
		err = io.EOF
	}

	return size, err
}

func InitSamplingReaderWriter(quota int64, goodThru time.Time) *SamplingReaderWriter {
	s := &SamplingReaderWriter{}

	s.SizeRead = 0
	s.SizeWritten = 0
	s.Events = []*IOEvent{}
	s.Quota = quota
	s.GoodThru = goodThru

	return s
}
