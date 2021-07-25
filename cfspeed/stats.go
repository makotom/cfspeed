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

func getDurationStats(durationSeries *[]time.Duration) *Stats {
	durationsMSF64 := []float64{}

	for _, duration := range *durationSeries {
		durationsMSF64 = append(durationsMSF64, float64(duration.Milliseconds()))
	}

	mean := getMean(&durationsMSF64)
	stderr := getStdErrUsingMean(&durationsMSF64, mean)

	return &Stats{
		mean,
		stderr,
	}
}

func getSpeedMeasurementStats(measurements *[]*SpeedMeasurement) *Stats {
	speedSamplesMBPS := []float64{}

	for _, measurement := range *measurements {
		speedSamplesMBPS = append(speedSamplesMBPS, float64(8*measurement.size/1000/1000)/measurement.duration.Seconds())
	}

	mean := getMean(&speedSamplesMBPS)
	stderr := getStdErrUsingMean(&speedSamplesMBPS, mean)

	return &Stats{
		mean,
		stderr,
	}
}
