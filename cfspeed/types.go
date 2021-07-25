package cfspeed

import "time"

type MeasurementMetadata struct {
	srcIP      string
	srcASN     string
	srcCity    string
	srcCountry string
	dstColo    string
}

type Stats struct {
	mean   float64
	stderr float64
}

type RTTStats struct {
	nSamples int
	mean     float64
	stderr   float64
}

type SpeedMeasurement struct {
	size     int64
	duration time.Duration
}

type SpeedMeasurementStats struct {
	nSamples int
	txSize   int64
	mean     float64
	stderr   float64
}
