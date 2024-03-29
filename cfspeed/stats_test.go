package cfspeed

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestGetStats_11Samples(t *testing.T) {
	samples := []float64{0.0, -0.5, 0.5, -1.0, 1.0, -1.5, 1.5, -2.0, 2.0, -2.5, 2.5}

	stats := getF64Stats(samples)

	assert.Equal(t, stats.NSamples, 11)
	assert.Equal(t, stats.Mean, 0.0)
	assert.Equal(t, stats.StdDev, 1.5811388300841898)
	assert.Equal(t, stats.StdErr, 0.4767312946227962)
	assert.Equal(t, stats.Min, -2.5)
	assert.Equal(t, stats.MinIndex, 9)
	assert.Equal(t, stats.Max, 2.5)
	assert.Equal(t, stats.MaxIndex, 10)
	assert.DeepEqual(t, stats.Deciles, []float64{-2.0, -1.5, -1.0, -0.5, 0.0, 0.5, 1.0, 1.5, 2.0})
}

func TestGetStats_6Samples(t *testing.T) {
	samples := []float64{-2.0, -3.0, 0.0, 2.0, -1.0, 1.0}

	stats := getF64Stats(samples)

	assert.Equal(t, stats.NSamples, 6)
	assert.Equal(t, stats.Mean, -0.5)
	assert.Equal(t, stats.StdDev, 1.707825127659933)
	assert.Equal(t, stats.StdErr, 0.6972166887783964)
	assert.Equal(t, stats.Min, -3.0)
	assert.Equal(t, stats.MinIndex, 1)
	assert.Equal(t, stats.Max, 2.0)
	assert.Equal(t, stats.MaxIndex, 3)
	assert.DeepEqual(t, stats.Deciles, []float64{-2.0, -2.0, -1.0, -1.0, 0.0, 0.0, 1.0, 1.0, 2.0})
}

func TestGetStats_25Samples(t *testing.T) {
	samples := []float64{127, 19, 139, 34, 134, 236, 221, 61, 146, 151, 157, 45, 137, 231, 46, 61, 215, 29, 189, 42, 108, 174, 235, 79, 167}

	stats := getF64Stats(samples)

	assert.Equal(t, stats.NSamples, 25)
	assert.Equal(t, stats.Mean, 127.31999999999996)
	assert.Equal(t, stats.StdDev, 70.00726819409546)
	assert.Equal(t, stats.StdErr, 14.001453638819092)
	assert.Equal(t, stats.Min, 19.0)
	assert.Equal(t, stats.MinIndex, 1)
	assert.Equal(t, stats.Max, 236.0)
	assert.Equal(t, stats.MaxIndex, 5)
	assert.DeepEqual(t, stats.Deciles, []float64{34, 46, 61, 127, 137, 146, 167, 189, 231})
}

func TestGetDurationStats(t *testing.T) {
	samples := []time.Duration{}

	for _, durationMS := range []int64{127, 19, 139, 34, 134, 236, 221, 61, 146, 151, 157, 45, 137, 231, 46, 61, 215, 29, 189, 42, 108, 174, 235, 79, 167} {
		samples = append(samples, time.Duration(durationMS*1000*1000))
	}

	stats := getDurationMSStats(samples)

	assert.Equal(t, stats.NSamples, 25)
	assert.Equal(t, stats.Mean, 127.31999999999996)
	assert.Equal(t, stats.StdDev, 70.00726819409546)
	assert.Equal(t, stats.StdErr, 14.001453638819092)
	assert.Equal(t, stats.Min, 19.0)
	assert.Equal(t, stats.MinIndex, 1)
	assert.Equal(t, stats.Max, 236.0)
	assert.Equal(t, stats.MaxIndex, 5)
	assert.DeepEqual(t, stats.Deciles, []float64{34, 46, 61, 127, 137, 146, 167, 189, 231})
}
