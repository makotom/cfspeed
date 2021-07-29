package cfspeed

import (
	"io"
	"time"
)

type IOEvent struct {
	Timestamp time.Time
	RW        string
	Size      int
}

type IOSampler struct {
	RW         string
	CallEvents []IOEvent
}

type ReadSampler struct {
	IOSampler

	ctr  int64
	cEOF int64
}

func InitReadSampler(size int64) *ReadSampler {
	r := &ReadSampler{}

	r.IOSampler.RW = "read"
	r.IOSampler.CallEvents = []IOEvent{}
	r.ctr = 0
	r.cEOF = size

	return r
}

func (r *ReadSampler) Read(p []byte) (int, error) {
	var err error = nil
	size := len(p)
	size64 := int64(size)

	r.ctr += size64

	if r.ctr > r.cEOF {
		size = int(size64 - (r.ctr - r.cEOF))
		err = io.EOF
	}

	r.IOSampler.CallEvents = append(r.IOSampler.CallEvents, IOEvent{
		Timestamp: time.Now(),
		Size:      size,
	})

	return size, err
}

type WriteSampler struct {
	IOSampler
}

func InitWriteSampler() *WriteSampler {
	w := &WriteSampler{}

	w.IOSampler.RW = "write"
	w.IOSampler.CallEvents = []IOEvent{}

	return w
}

func (w *WriteSampler) Write(p []byte) (int, error) {
	size := len(p)

	w.IOSampler.CallEvents = append(w.IOSampler.CallEvents, IOEvent{
		Timestamp: time.Now(),
		RW:        "write",
		Size:      size,
	})

	return size, nil
}
