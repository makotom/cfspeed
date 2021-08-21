package cfspeed

import (
	"math"
	"sort"
	"time"
)

const ioSamplingWindowWidthMin = 200 * time.Millisecond

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

func getStdDevUsingMean(series []float64, mean float64) float64 {
	return math.Sqrt(getSquareMean(series) - (mean * mean))
}

func getDeciles(series []float64) []float64 {
	ret := make([]float64, 9)
	sorted := make([]float64, len(series))
	elemsPerStep := float64(len(series)) / 10

	if len(series) == 0 {
		return ret
	}

	if copy(sorted, series) == 0 {
		return ret
	}

	sort.Float64s(sorted)

	for iter := 1; iter < 10; iter += 1 {
		ret[iter-1] = sorted[int64(math.Floor(elemsPerStep*float64(iter)))]
	}

	return ret
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
	ret.StdDev = getStdDevUsingMean(series, ret.Mean)
	ret.StdErr = ret.StdDev / math.Sqrt(float64(len(series)))
	ret.Deciles = getDeciles(series)

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

func reverseF64sInPlace(series []float64) {
	seriesLen := len(series)
	halfLen := seriesLen / 2

	for iter := 0; iter < halfLen; iter += 1 {
		series[iter], series[seriesLen-1-iter] = series[seriesLen-1-iter], series[iter]
	}
}

func getIOLatencyStats(ioEvents []*IOEvent) *Stats {
	latencies := []time.Duration{}

	for index, event := range ioEvents[1:] {
		latencies = append(latencies, event.Timestamp.Sub(ioEvents[index].Timestamp))
	}

	return getDurationStats(latencies)
}

func analyseIOReadEvents(_, end time.Time, cfReqDur time.Duration, ioEvents []*IOEvent) []float64 {
	mbpsSamples := []float64{}

	ioLatencyStats := getIOLatencyStats(ioEvents)
	samplingBoundaryLatencyThreshold := ioLatencyStats.Mean + 2*ioLatencyStats.StdDev

	ioEventsToAnalyse := ioEvents
	adjustedEndTime := end.Add(-cfReqDur)
	if adjustedEndTime.Sub(ioEvents[len(ioEvents)-1].Timestamp) > 0 {
		ioEventsToAnalyse = append(ioEvents, &IOEvent{
			Timestamp: adjustedEndTime,
			Mode:      "read",
			Size:      0,
		})
	}

	windowStart := ioEventsToAnalyse[0].Timestamp
	sizeSum := 0
	for index, event := range ioEventsToAnalyse[1:] {
		sizeSum += ioEventsToAnalyse[index].Size

		sinceStart := event.Timestamp.Sub(windowStart)
		if float64(event.Timestamp.Sub(ioEventsToAnalyse[index].Timestamp).Milliseconds()) > samplingBoundaryLatencyThreshold && sinceStart > ioSamplingWindowWidthMin {
			mbpsSamples = append(mbpsSamples, float64(8*sizeSum)/float64(sinceStart.Microseconds()))

			windowStart = event.Timestamp
			sizeSum = 0
		}
	}

	return mbpsSamples
}

func analyseIOWriteEvents(start, _ time.Time, cfReqDur time.Duration, ioEvents []*IOEvent) []float64 {
	mbpsSamples := []float64{}

	ioLatencyStats := getIOLatencyStats(ioEvents)
	samplingBoundaryLatencyThreshold := ioLatencyStats.Mean + 2*ioLatencyStats.StdDev

	ioEventsToAnalyse := getReversedIOEvents(ioEvents)
	adjustedStartTime := start.Add(cfReqDur)
	if ioEvents[0].Timestamp.Sub(adjustedStartTime) > 0 {
		ioEventsToAnalyse = append(ioEventsToAnalyse, &IOEvent{
			Timestamp: adjustedStartTime,
			Mode:      "write",
			Size:      0,
		})
	}

	windowStart := ioEventsToAnalyse[0].Timestamp
	sizeSum := 0
	for index, event := range ioEventsToAnalyse[1:] {
		sizeSum += ioEventsToAnalyse[index].Size

		sinceStart := windowStart.Sub(event.Timestamp)
		if float64(ioEventsToAnalyse[index].Timestamp.Sub(event.Timestamp).Milliseconds()) > samplingBoundaryLatencyThreshold && sinceStart > ioSamplingWindowWidthMin {
			mbpsSamples = append(mbpsSamples, float64(8*sizeSum)/float64(sinceStart.Microseconds()))

			windowStart = event.Timestamp
			sizeSum = 0
		}
	}

	reverseF64sInPlace(mbpsSamples)

	return mbpsSamples
}

func analyseIOEvents(ioMode string, start, end time.Time, cfReqDurStats *Stats, ioEvents []*IOEvent) []float64 {
	cfReqDur := time.Duration(cfReqDurStats.Mean * 1000 * 1000)

	switch ioMode {
	case "read":
		return analyseIOReadEvents(start, end, cfReqDur, ioEvents)
	case "write":
		return analyseIOWriteEvents(start, end, cfReqDur, ioEvents)
	default:
		return []float64{}
	}
}

func getDurationStats(durations []time.Duration) *Stats {
	durationSamples := []float64{}

	for _, duration := range durations {
		durationMSF64 := float64(duration.Nanoseconds()) / 1000 / 1000
		durationSamples = append(durationSamples, durationMSF64)
	}

	return getStats(durationSamples)
}

func getSpeedMeasurementStats(measurements []*SpeedMeasurement, cfReqDurStats *Stats) (float64, *Stats) {
	mbpsSamples := []float64{}
	sizeSum := int64(0)
	durationSum := int64(0)

	for _, measurement := range measurements {
		mbpsSamples = append(mbpsSamples, analyseIOEvents(measurement.IOSampler.Mode, measurement.Start, measurement.End, cfReqDurStats, measurement.IOSampler.CallEvents)...)
		sizeSum += measurement.Size
		durationSum += measurement.Duration.Microseconds()
	}

	return float64(8*sizeSum) / float64(durationSum), getStats(mbpsSamples)
}
