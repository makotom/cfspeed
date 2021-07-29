package cfspeed

import (
	"math"
	"time"
)

const (
	ioSamplingWindowWidthMin = 50 * time.Millisecond
	ioSamplingWindowWidthMax = 250 * time.Millisecond
)

func getMean(series *[]float64) float64 {
	ret := float64(0)
	nSamplesF64 := float64(len(*series))

	for _, element := range *series {
		ret += element / nSamplesF64
	}

	return ret
}

func getSquareMean(series *[]float64) float64 {
	ret := float64(0)
	nSamplesF64 := float64(len(*series))

	for _, element := range *series {
		ret += element * element / nSamplesF64
	}

	return ret
}

func getStdErrUsingMean(series *[]float64, mean float64) float64 {
	return math.Sqrt(getSquareMean(series)-(mean*mean)) / math.Sqrt(float64(len(*series)))
}

func getStats(series *[]float64) *Stats {
	ret := &Stats{
		Min:      math.Inf(1),
		Max:      math.Inf(-1),
		MinIndex: 0,
		MaxIndex: 0,
	}

	for index, element := range *series {
		if element < ret.Min {
			ret.Min = element
			ret.MinIndex = index
		}
		if element > ret.Max {
			ret.Max = element
			ret.MaxIndex = index
		}
	}

	ret.NSamples = len(*series)
	ret.Mean = getMean(series)
	ret.StdErr = getStdErrUsingMean(series, ret.Mean)

	return ret
}

func analyseIOEvents(ioEvents *[]IOEvent, ioRW string) *[]float64 {
	mbpsSamples := []float64{}

	windowStart := (*ioEvents)[0].Timestamp
	sizeSum := 0

	for index, event := range *ioEvents {
		if index == 0 {
			continue
		} else {
			if ioRW == "read" {
				sizeSum += (*ioEvents)[index-1].Size
			} else {
				sizeSum += event.Size
			}

			sinceStart := event.Timestamp.Sub(windowStart)
			if sinceStart > ioSamplingWindowWidthMax {
				mbpsSamples = append(mbpsSamples, float64(8*sizeSum)/float64(sinceStart.Microseconds()))

				windowStart = event.Timestamp
				sizeSum = 0
			}
		}
	}

	sinceStart := (*ioEvents)[len(*ioEvents)-1].Timestamp.Sub(windowStart)
	if sinceStart > ioSamplingWindowWidthMin {
		mbpsSamples = append(mbpsSamples, float64(8*sizeSum)/float64(sinceStart.Microseconds()))
	}

	return &mbpsSamples
}

func getDurationStats(durations *[]time.Duration) *Stats {
	durationSamples := []float64{}

	for _, duration := range *durations {
		durationMSF64 := float64(duration.Microseconds()) / 1000
		durationSamples = append(durationSamples, durationMSF64)
	}

	return getStats(&durationSamples)
}

func getSpeedMeasurementStats(measurements *[]*SpeedMeasurement) *Stats {
	mbpsSamples := []float64{}

	for _, measurement := range *measurements {
		mbpsSamples = append(mbpsSamples, *analyseIOEvents(&measurement.IOSampler.CallEvents, measurement.IOSampler.RW)...)
	}

	return getStats(&mbpsSamples)
}
