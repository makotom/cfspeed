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
	Mode       string
	CallEvents []*IOEvent
}

type ReadSampler struct {
	IOSampler

	ctr  int64
	cEOF int64
}

func InitReadSampler(size int64) *ReadSampler {
	r := &ReadSampler{}

	r.IOSampler.Mode = "read"
	r.IOSampler.CallEvents = []*IOEvent{}
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
		r.ctr = r.cEOF
	}

	r.IOSampler.CallEvents = append(r.IOSampler.CallEvents, &IOEvent{
		Timestamp: time.Now(),
		Mode:      "read",
		Size:      size,
	})

	return size, err
}

type WriteSampler struct {
	IOSampler
}

func InitWriteSampler() *WriteSampler {
	w := &WriteSampler{}

	w.IOSampler.Mode = "write"
	w.IOSampler.CallEvents = []*IOEvent{}

	return w
}

func (w *WriteSampler) Write(p []byte) (int, error) {
	size := len(p)

	w.IOSampler.CallEvents = append(w.IOSampler.CallEvents, &IOEvent{
		Timestamp: time.Now(),
		Mode:      "write",
		Size:      size,
	})

	return size, nil
}
