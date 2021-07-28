package cfspeed

import (
	"net/http"
	"time"
)

type MeasurementMetadata struct {
	SrcIP      string
	SrcASN     string
	SrcCity    string
	SrcCountry string
	DstColo    string
}

type Stats struct {
	NSamples int
	Mean     float64
	StdErr   float64
	Min      float64
	Max      float64
}

type SpeedMeasurement struct {
	Size           int64
	Duration       time.Duration
	HTTPRespHeader http.Header
}

type SpeedMeasurementStats struct {
	NSamples int
	TXSize   int64
	Mean     float64
	StdErr   float64
	Min      float64
	Max      float64
}
