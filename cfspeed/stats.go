package cfspeed

import (
	"math"
	"time"
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
		Min: math.Inf(1),
		Max: math.Inf(-1),
	}

	for _, element := range *series {
		if element < ret.Min {
			ret.Min = element
		}
		if element > ret.Max {
			ret.Max = element
		}
	}

	ret.NSamples = len(*series)
	ret.Mean = getMean(series)
	ret.StdErr = getStdErrUsingMean(series, ret.Mean)

	return ret
}

func getDurationStats(durations *[]time.Duration) *Stats {
	durationSamples := []float64{}

	for _, duration := range *durations {
		durationMSF64 := float64(duration.Milliseconds())
		durationSamples = append(durationSamples, durationMSF64)
	}

	return getStats(&durationSamples)
}

func getSpeedMeasurementStats(measurements *[]*SpeedMeasurement) *Stats {
	mbpsSamples := []float64{}

	for _, measurement := range *measurements {
		mbps := float64(8*measurement.Size) / float64(measurement.Duration.Microseconds())
		mbpsSamples = append(mbpsSamples, mbps)
	}

	return getStats(&mbpsSamples)
}
