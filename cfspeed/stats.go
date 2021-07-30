package cfspeed

import (
	"math"
	"time"
)

const ioSamplingWindowWidth = 100 * time.Millisecond

func getMean(series []float64) float64 {
	ret := float64(0)
	nSamplesF64 := float64(len(series))

	for _, element := range series {
		ret += element / nSamplesF64
	}

	return ret
}

func getSquareMean(series []float64) float64 {
	ret := float64(0)
	nSamplesF64 := float64(len(series))

	for _, element := range series {
		ret += element * element / nSamplesF64
	}

	return ret
}

func getStdErrUsingMean(series []float64, mean float64) float64 {
	return math.Sqrt(getSquareMean(series)-(mean*mean)) / math.Sqrt(float64(len(series)))
}

func getStats(series []float64) *Stats {
	ret := &Stats{
		Min:      math.Inf(1),
		Max:      math.Inf(-1),
		MinIndex: 0,
		MaxIndex: 0,
	}

	for index, element := range series {
		if element < ret.Min {
			ret.Min = element
			ret.MinIndex = index
		}
		if element > ret.Max {
			ret.Max = element
			ret.MaxIndex = index
		}
	}

	ret.NSamples = len(series)
	ret.Mean = getMean(series)
	ret.StdErr = getStdErrUsingMean(series, ret.Mean)

	return ret
}

func getReversedIOEvents(ioEvents []*IOEvent) []*IOEvent {
	ret := []*IOEvent{}
	seriesLen := len(ioEvents)

	for iter := 0; iter < seriesLen; iter += 1 {
		ret = append(ret, ioEvents[seriesLen-1-iter])
	}

	return ret
}

func reverseF64InPlace(series []float64) {
	seriesLen := len(series)
	halfLen := seriesLen / 2

	for iter := 0; iter < halfLen; iter += 1 {
		series[iter], series[seriesLen-1-iter] = series[seriesLen-1-iter], series[iter]
	}
}

func analyseIOEvents(ioEvents []*IOEvent, ioMode string) []float64 {
	mbpsSamples := []float64{}

	ioEventsReversed := getReversedIOEvents(ioEvents)

	windowStart := ioEventsReversed[0].Timestamp
	sizeSum := 0
	for index, event := range ioEventsReversed[1:] {
		if ioMode == "read" {
			sizeSum += event.Size
		} else {
			sizeSum += ioEventsReversed[index].Size
		}

		sinceStart := windowStart.Sub(event.Timestamp)
		if sinceStart > ioSamplingWindowWidth {
			mbpsSamples = append(mbpsSamples, float64(8*sizeSum)/float64(sinceStart.Microseconds()))

			windowStart = event.Timestamp
			sizeSum = 0
		}
	}

	reverseF64InPlace(mbpsSamples)

	return mbpsSamples
}

func getDurationStats(durations []time.Duration) *Stats {
	durationSamples := []float64{}

	for _, duration := range durations {
		durationMSF64 := float64(duration.Microseconds()) / 1000
		durationSamples = append(durationSamples, durationMSF64)
	}

	return getStats(durationSamples)
}

func getSpeedMeasurementStats(measurements []*SpeedMeasurement) (float64, *Stats) {
	mbpsSamples := []float64{}
	sizeSum := int64(0)
	durationSum := int64(0)

	for _, measurement := range measurements {
		mbpsSamples = append(mbpsSamples, analyseIOEvents(measurement.IOSampler.CallEvents, measurement.IOSampler.Mode)...)
		sizeSum += measurement.Size
		durationSum += measurement.Duration.Microseconds()
	}

	return float64(8*sizeSum) / float64(durationSum), getStats(mbpsSamples)
}
