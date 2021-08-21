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
	StdDev   float64
	StdErr   float64
	Min      float64
	MinIndex int
	Max      float64
	MaxIndex int
	Deciles  []float64
}

type SpeedMeasurement struct {
	Size           int64
	Start          time.Time
	End            time.Time
	Duration       time.Duration
	IOSampler      IOSampler
	HTTPRespHeader http.Header
}

type SpeedMeasurementStats struct {
	NSamples int
	TXSize   int64
	NTX      int
	Mean     float64
	StdErr   float64
	Min      float64
	Max      float64
	Deciles  []float64
	CatSpeed float64
}
