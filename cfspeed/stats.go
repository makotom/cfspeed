package cfspeed

import (
	"math"
	"sort"
	"time"
)

const (
	ioSamplingWindowWidthMin = 100 * time.Millisecond
)

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

type Sample[T any] struct {
	Value     T
	Timestamp time.Time
}

func sumF64s(values []float64) float64 {
	ret := float64(0)

	for _, value := range values {
		ret += value
	}

	return ret
}

func getF64Mean(series []float64) float64 {
	ret := float64(0)
	nSamplesF64 := float64(len(series))

	for _, element := range series {
		ret += element / nSamplesF64
	}

	return ret
}

func getF64SquareMean(series []float64) float64 {
	ret := float64(0)
	nSamplesF64 := float64(len(series))

	for _, element := range series {
		ret += element * element / nSamplesF64
	}

	return ret
}

func getF64StdDevUsingMean(series []float64, mean float64) float64 {
	return math.Sqrt(getF64SquareMean(series) - (mean * mean))
}

func getF64Deciles(series []float64) []float64 {
	ret := make([]float64, 9)
	sorted := make([]float64, len(series))
	elemsPerStep := float64(len(series)-1) / 10

	if len(series) == 0 {
		return ret
	}

	if copy(sorted, series) == 0 {
		return ret
	}

	sort.Float64s(sorted)

	for iter := 1; iter < 10; iter += 1 {
		ret[iter-1] = sorted[int64(math.Round(elemsPerStep*float64(iter)))]
	}

	return ret
}

func getF64Stats(series []float64) *Stats {
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
	ret.Mean = getF64Mean(series)
	ret.StdDev = getF64StdDevUsingMean(series, ret.Mean)
	ret.StdErr = ret.StdDev / math.Sqrt(float64(len(series)))
	ret.Deciles = getF64Deciles(series)

	return ret
}

func reverseValueSamplesInPlace[T any](series []*Sample[T]) {
	seriesLen := len(series)
	halfLen := seriesLen / 2

	for iter := 0; iter < halfLen; iter += 1 {
		series[iter], series[seriesLen-1-iter] = series[seriesLen-1-iter], series[iter]
	}
}

func getValuesFromSamples[T any](samples []*Sample[T]) []T {
	ret := make([]T, len(samples))

	for index, sample := range samples {
		ret[index] = sample.Value
	}

	return ret
}

func getDurationMSStats(durations []time.Duration) *Stats {
	durationSamples := make([]float64, len(durations))

	for index, duration := range durations {
		durationMSF64 := float64(duration.Nanoseconds()) / 1000 / 1000
		durationSamples[index] = durationMSF64
	}

	return getF64Stats(durationSamples)
}

func getIOLatencyMSStats(ioEvents []*IOEvent) *Stats {
	latencies := make([]time.Duration, len(ioEvents)-1)

	for index, event := range ioEvents[1:] {
		latencies[index] = event.Timestamp.Sub(ioEvents[index].Timestamp)
	}

	return getDurationMSStats(latencies)
}

func getReversedIOEvents(ioEvents []*IOEvent) []*IOEvent {
	ret := make([]*IOEvent, len(ioEvents))
	seriesLen := len(ioEvents)

	for iter := 0; iter < seriesLen; iter += 1 {
		ret[iter] = ioEvents[seriesLen-1-iter]
	}

	return ret
}
func getIOReadMBPSSamples(_, end time.Time, cfReqDur time.Duration, ioEvents []*IOEvent) []*Sample[float64] {
	mbpsSamples := []*Sample[float64]{}

	ioEventsToAnalyse := ioEvents
	adjustedEndTime := end.Add(-cfReqDur)
	if adjustedEndTime.Compare(ioEvents[len(ioEvents)-1].Timestamp) > 0 {
		ioEventsToAnalyse = append(ioEvents, &IOEvent{
			Timestamp: adjustedEndTime,
			Mode:      IOModeRead,
			Size:      0,
		})
	}

	ioLatencyMSStats := getIOLatencyMSStats(ioEventsToAnalyse)
	inferredUnbufferedIOThreshold := ioLatencyMSStats.Mean + 2*ioLatencyMSStats.StdDev

	windowStart := ioEventsToAnalyse[0].Timestamp
	lastIndexForFor := len(ioEventsToAnalyse) - 1 - 1
	sizeSum := 0
	for index, event := range ioEventsToAnalyse[1:] {
		sizeSum += ioEventsToAnalyse[index].Size

		sinceStart := event.Timestamp.Sub(windowStart)
		if (float64(event.Timestamp.Sub(ioEventsToAnalyse[index].Timestamp).Milliseconds()) > inferredUnbufferedIOThreshold && sinceStart > ioSamplingWindowWidthMin) || index == lastIndexForFor {
			mbpsSamples = append(mbpsSamples, &Sample[float64]{
				Value:     float64(8*sizeSum) / float64(sinceStart.Microseconds()),
				Timestamp: event.Timestamp,
			})

			windowStart = event.Timestamp
			sizeSum = 0
		}
	}

	return mbpsSamples
}

func getIOWriteMBPSSamples(start, _ time.Time, cfReqDur time.Duration, ioEvents []*IOEvent) []*Sample[float64] {
	mbpsSamples := []*Sample[float64]{}

	ioEventsToAnalyse := getReversedIOEvents(ioEvents)
	adjustedStartTime := start.Add(cfReqDur)
	if ioEvents[0].Timestamp.Compare(adjustedStartTime) > 0 {
		ioEventsToAnalyse = append(ioEventsToAnalyse, &IOEvent{
			Timestamp: adjustedStartTime,
			Mode:      IOModeWrite,
			Size:      0,
		})
	}

	ioLatencyMSStats := getIOLatencyMSStats(ioEventsToAnalyse)
	inferredUnbufferedIOThreshold := math.Abs(ioLatencyMSStats.Mean) + 2*ioLatencyMSStats.StdDev

	windowStart := ioEventsToAnalyse[0].Timestamp
	lastIndexForFor := len(ioEventsToAnalyse) - 1 - 1
	sizeSum := 0
	for index, event := range ioEventsToAnalyse[1:] {
		sizeSum += ioEventsToAnalyse[index].Size

		sinceStart := windowStart.Sub(event.Timestamp)
		if (float64(ioEventsToAnalyse[index].Timestamp.Sub(event.Timestamp).Milliseconds()) > inferredUnbufferedIOThreshold && sinceStart > ioSamplingWindowWidthMin) || index == lastIndexForFor {
			mbpsSamples = append(mbpsSamples, &Sample[float64]{
				Value:     float64(8*sizeSum) / float64(sinceStart.Microseconds()),
				Timestamp: windowStart,
			})

			windowStart = event.Timestamp
			sizeSum = 0
		}
	}

	reverseValueSamplesInPlace(mbpsSamples)

	return mbpsSamples
}

func getMBPSSamplesFromMeasurement(measurement *SpeedMeasurement) []*Sample[float64] {
	switch measurement.Direction {
	case DirectionDownlink:
		return getIOWriteMBPSSamples(measurement.Start, measurement.End, measurement.CFReqDur, measurement.IOSampler.Events)
	case DirectionUplink:
		return getIOReadMBPSSamples(measurement.Start, measurement.End, measurement.CFReqDur, measurement.IOSampler.Events)
	default:
		return []*Sample[float64]{}
	}
}

func chooseGroupWithYoungestHead(groupedMBPSSamples [][]*Sample[float64], groupHead []int) int {
	youngestGroup := -1

	youngestTimestamp := time.Time{}

	for index := range groupHead {
		if groupHead[index] > -1 {
			if groupedMBPSSamples[index][groupHead[index]].Timestamp.Compare(youngestTimestamp) > 0 {
				youngestGroup = index
				youngestTimestamp = groupedMBPSSamples[index][groupHead[index]].Timestamp
			}
		}
	}

	return youngestGroup
}

func consolidateGroupedMBPSSamples(groupedMBPSSamples [][]*Sample[float64]) []*Sample[float64] {
	nGroups := len(groupedMBPSSamples)

	mergedMBPSSamples := []*Sample[float64]{}
	curGroupMBPS := make([]float64, nGroups)
	curGroupHead := make([]int, nGroups)

	for iter := 0; iter < nGroups; iter += 1 {
		curGroupHead[iter] = len(groupedMBPSSamples[iter]) - 1
	}

	for {
		youngestGroup := chooseGroupWithYoungestHead(groupedMBPSSamples, curGroupHead)
		if youngestGroup < 0 {
			break
		}

		youngestSample := groupedMBPSSamples[youngestGroup][curGroupHead[youngestGroup]]
		curGroupMBPS[youngestGroup] = youngestSample.Value

		// Add the sample if and only if there is one or more active groups
		if curSummedMBPS := sumF64s(curGroupMBPS); curSummedMBPS > 0 {
			mergedMBPSSamples = append(mergedMBPSSamples, &Sample[float64]{
				Value:     curSummedMBPS,
				Timestamp: youngestSample.Timestamp,
			})
		}

		curGroupHead[youngestGroup] -= 1
	}

	reverseValueSamplesInPlace[float64](mergedMBPSSamples)

	return mergedMBPSSamples
}

func analyseMeasurements(measurements []*SpeedMeasurement, injectZeroPointSample bool) ([]*Sample[float64], int64, int64) {
	mbpsSamples := []*Sample[float64]{}
	sizeSum := int64(0)
	durationSum := int64(0)

	for _, measurement := range measurements {
		if injectZeroPointSample {
			switch measurement.Direction {
			case DirectionDownlink:
				mbpsSamples = append(mbpsSamples, &Sample[float64]{
					Value:     0,
					Timestamp: measurement.Start.Add(measurement.CFReqDur),
				})
			case DirectionUplink:
				mbpsSamples = append(mbpsSamples, &Sample[float64]{
					Value:     0,
					Timestamp: measurement.Start,
				})
			}
		}

		mbpsSamples = append(mbpsSamples, getMBPSSamplesFromMeasurement(measurement)...)

		sizeSum += measurement.Size
		durationSum += measurement.Duration.Microseconds()
	}

	return mbpsSamples, sizeSum, durationSum
}

func analyseMeasurementGroups(measurementGroups [][]*SpeedMeasurement) ([]*Sample[float64], int64, int64) {
	sizeSum := int64(0)

	groupedMBPSSamples := make([][]*Sample[float64], len(measurementGroups))
	firstStart := time.Unix(1<<62-1, 1<<62-1)
	lastEnd := time.Time{}

	for index, measurements := range measurementGroups {
		groupMBPSSamples, groupSizeSum, _ := analyseMeasurements(measurements, true)
		groupedMBPSSamples[index] = groupMBPSSamples
		sizeSum += groupSizeSum

		firstMeasurement := measurements[0]
		if firstMeasurement.Start.Compare(firstStart) < 0 {
			firstStart = firstMeasurement.Start
		}

		lastMeasurement := measurements[len(measurements)-1]
		if lastMeasurement.End.Compare(lastEnd) > 0 {
			lastEnd = lastMeasurement.End
		}
	}

	return consolidateGroupedMBPSSamples(groupedMBPSSamples), sizeSum, lastEnd.Sub(firstStart).Microseconds()
}

func getSingleSpeedMeasurementStats(measurements []*SpeedMeasurement) (*Stats, int64, int64) {
	mbpsSamples, sizeSum, durationSum := analyseMeasurements(measurements, false)
	return getF64Stats(getValuesFromSamples(mbpsSamples)), sizeSum, durationSum
}

func getMultiplexedSpeedMeasurementStats(measurementGroups [][]*SpeedMeasurement) (*Stats, int64, int64) {
	mbpsSamples, sizeSum, longestSpan := analyseMeasurementGroups(measurementGroups)
	return getF64Stats(getValuesFromSamples(mbpsSamples)), sizeSum, longestSpan
}
