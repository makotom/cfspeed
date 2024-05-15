package cfspeed

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestGetF64Stats_11Samples(t *testing.T) {
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

func TestGetF64Stats_6Samples(t *testing.T) {
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

func TestGetF64Stats_25Samples(t *testing.T) {
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

func TestGetDurationMSStats(t *testing.T) {
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

func generateDummyIOEvents(ioMode string, startAt time.Time, eventsAfter []time.Duration, eventSizes []int) []*IOEvent {
	return []*IOEvent{
		{
			Timestamp: startAt.Add(eventsAfter[0]),
			Mode:      ioMode,
			Size:      eventSizes[0],
		},
		// Below inject null events with the same timestamp in order to lower inferredUnbufferedIOThreshold
		// Inject 6 times to yield (mean + 2 * stddev) that is "slightly" less than 500 for both sets of {200, 300, 500} and {100 500 400}
		{
			Timestamp: startAt.Add(eventsAfter[0]),
			Mode:      ioMode,
			Size:      0,
		},
		{
			Timestamp: startAt.Add(eventsAfter[0]),
			Mode:      ioMode,
			Size:      0,
		},
		{
			Timestamp: startAt.Add(eventsAfter[0]),
			Mode:      ioMode,
			Size:      0,
		},
		{
			Timestamp: startAt.Add(eventsAfter[0]),
			Mode:      ioMode,
			Size:      0,
		},
		{
			Timestamp: startAt.Add(eventsAfter[0]),
			Mode:      ioMode,
			Size:      0,
		},
		{
			Timestamp: startAt.Add(eventsAfter[0]),
			Mode:      ioMode,
			Size:      0,
		},
		// Zero-fill above
		{
			Timestamp: startAt.Add(eventsAfter[1]),
			Mode:      ioMode,
			Size:      eventSizes[1],
		},
		{
			Timestamp: startAt.Add(eventsAfter[2]),
			Mode:      ioMode,
			Size:      eventSizes[2],
		},
	}
}

func TestAnalyseMeasurements_DLWithoutZeroPoint(t *testing.T) {
	dummyMeasurementSize := int64(50 * 1000 * 1000) // 400 MBit
	dummyIOSizes := []int{
		5 * 1000 * 1000,  // 40 Mbit, 200 Mbps
		15 * 1000 * 1000, // 120 Mbit, 400 Mbps
		30 * 1000 * 1000, // 240 MBit, 480 Mbps
	}

	dummyCFReqDur := 20 * time.Millisecond
	dummyDuration := 1000*time.Millisecond + dummyCFReqDur
	dummyIOEventsAfter := []time.Duration{
		200*time.Millisecond + dummyCFReqDur,
		500*time.Millisecond + dummyCFReqDur,
		dummyDuration,
	}

	dummyFirstStart := time.Now()
	dummySecondStart := dummyFirstStart.Add(2 * time.Second)

	dummyMeasurements := []*SpeedMeasurement{
		{
			Direction: DirectionDownlink,
			Size:      dummyMeasurementSize,
			Start:     dummyFirstStart,
			End:       dummyFirstStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeWrite, dummyFirstStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
		{
			Direction: DirectionDownlink,
			Size:      dummyMeasurementSize,
			Start:     dummySecondStart,
			End:       dummySecondStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeWrite, dummySecondStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
	}

	mbpsSamples, sizeSum, durationSum := analyseMeasurements(dummyMeasurements, false)

	assert.Equal(t, len(mbpsSamples), 4)
	assert.DeepEqual(t, *mbpsSamples[0], Sample[float64]{
		Value:     320,
		Timestamp: dummyFirstStart.Add(dummyIOEventsAfter[1]),
	})
	assert.DeepEqual(t, *mbpsSamples[1], Sample[float64]{
		Value:     480,
		Timestamp: dummyFirstStart.Add(dummyIOEventsAfter[2]),
	})
	assert.DeepEqual(t, *mbpsSamples[2], Sample[float64]{
		Value:     320,
		Timestamp: dummySecondStart.Add(dummyIOEventsAfter[1]),
	})
	assert.DeepEqual(t, *mbpsSamples[3], Sample[float64]{
		Value:     480,
		Timestamp: dummySecondStart.Add(dummyIOEventsAfter[2]),
	})

	assert.Equal(t, sizeSum, 2*dummyMeasurementSize)
	assert.Equal(t, time.Duration(durationSum)*time.Microsecond, 2*dummyDuration)
}

func TestAnalyseMeasurements_DLWithZeroPoint(t *testing.T) {
	dummyMeasurementSize := int64(50 * 1000 * 1000) // 400 MBit
	dummyIOSizes := []int{
		5 * 1000 * 1000,  // 40 Mbit, 200 Mbps
		15 * 1000 * 1000, // 120 Mbit, 400 Mbps
		30 * 1000 * 1000, // 240 MBit, 480 Mbps
	}

	dummyCFReqDur := 20 * time.Millisecond
	dummyDuration := 1000*time.Millisecond + dummyCFReqDur
	dummyIOEventsAfter := []time.Duration{
		200*time.Millisecond + dummyCFReqDur,
		500*time.Millisecond + dummyCFReqDur,
		dummyDuration,
	}

	dummyFirstStart := time.Now()
	dummySecondStart := dummyFirstStart.Add(2 * time.Second)

	dummyMeasurements := []*SpeedMeasurement{
		{
			Direction: DirectionDownlink,
			Size:      dummyMeasurementSize,
			Start:     dummyFirstStart,
			End:       dummyFirstStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeWrite, dummyFirstStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
		{
			Direction: DirectionDownlink,
			Size:      dummyMeasurementSize,
			Start:     dummySecondStart,
			End:       dummySecondStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeWrite, dummySecondStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
	}

	mbpsSamples, sizeSum, durationSum := analyseMeasurements(dummyMeasurements, true)

	assert.Equal(t, len(mbpsSamples), 6)
	assert.DeepEqual(t, *mbpsSamples[0], Sample[float64]{
		Value:     0,
		Timestamp: dummyFirstStart.Add(dummyCFReqDur),
	})
	assert.DeepEqual(t, *mbpsSamples[1], Sample[float64]{
		Value:     320,
		Timestamp: dummyFirstStart.Add(dummyIOEventsAfter[1]),
	})
	assert.DeepEqual(t, *mbpsSamples[2], Sample[float64]{
		Value:     480,
		Timestamp: dummyFirstStart.Add(dummyIOEventsAfter[2]),
	})
	assert.DeepEqual(t, *mbpsSamples[3], Sample[float64]{
		Value:     0,
		Timestamp: dummySecondStart.Add(dummyCFReqDur),
	})
	assert.DeepEqual(t, *mbpsSamples[4], Sample[float64]{
		Value:     320,
		Timestamp: dummySecondStart.Add(dummyIOEventsAfter[1]),
	})
	assert.DeepEqual(t, *mbpsSamples[5], Sample[float64]{
		Value:     480,
		Timestamp: dummySecondStart.Add(dummyIOEventsAfter[2]),
	})

	assert.Equal(t, sizeSum, 2*dummyMeasurementSize)
	assert.Equal(t, time.Duration(durationSum)*time.Microsecond, 2*dummyDuration)
}

func TestAnalyseMeasurements_DLWithCoincidentEvents(t *testing.T) {
	dummyMeasurementSize := int64(50 * 1000 * 1000) // 400 MBit
	dummyIOSizes := []int{
		0,
		20 * 1000 * 1000, // 160 Mbit
		30 * 1000 * 1000, // 240 MBit
	}

	dummyCFReqDur := 20 * time.Millisecond
	dummyDuration := 1000*time.Millisecond + dummyCFReqDur
	dummyIOEventsAfter := []time.Duration{
		dummyCFReqDur,
		dummyDuration,
		dummyDuration,
	}

	dummyStart := time.Now()

	dummyMeasurements := []*SpeedMeasurement{
		{
			Direction: DirectionDownlink,
			Size:      dummyMeasurementSize,
			Start:     dummyStart,
			End:       dummyStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeWrite, dummyStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
	}

	mbpsSamples, sizeSum, durationSum := analyseMeasurements(dummyMeasurements, false)

	assert.Equal(t, len(mbpsSamples), 1)
	assert.DeepEqual(t, *mbpsSamples[0], Sample[float64]{
		Value:     400,
		Timestamp: dummyStart.Add(dummyDuration),
	})

	assert.Equal(t, sizeSum, dummyMeasurementSize)
	assert.Equal(t, time.Duration(durationSum)*time.Microsecond, dummyDuration)
}

func TestAnalyseMeasurements_ULWithoutZeroPoint(t *testing.T) {
	dummyMeasurementSize := int64(54 * 1000 * 1000) // 432 MBit
	dummyIOSizes := []int{
		5 * 1000 * 1000,  // 20 Mbit, 400 Mbps
		25 * 1000 * 1000, // 220 Mbit, 440 Mbps
		24 * 1000 * 1000, // 192 MBit, 480 Mbps
	}

	dummyCFReqDur := 20 * time.Millisecond
	dummyDuration := 1000*time.Millisecond + dummyCFReqDur
	dummyIOEventsAfter := []time.Duration{
		0,
		100 * time.Millisecond,
		600 * time.Millisecond,
	}

	dummyFirstStart := time.Now()
	dummySecondStart := dummyFirstStart.Add(2 * time.Second)

	dummyMeasurements := []*SpeedMeasurement{
		{
			Direction: DirectionUplink,
			Size:      dummyMeasurementSize,
			Start:     dummyFirstStart,
			End:       dummyFirstStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeRead, dummyFirstStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
		{
			Direction: DirectionUplink,
			Size:      dummyMeasurementSize,
			Start:     dummySecondStart,
			End:       dummySecondStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeRead, dummySecondStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
	}

	mbpsSamples, sizeSum, durationSum := analyseMeasurements(dummyMeasurements, false)

	assert.Equal(t, len(mbpsSamples), 4)
	assert.DeepEqual(t, *mbpsSamples[0], Sample[float64]{
		Value:     400,
		Timestamp: dummyFirstStart.Add(dummyIOEventsAfter[2]),
	})
	assert.DeepEqual(t, *mbpsSamples[1], Sample[float64]{
		Value:     480,
		Timestamp: dummyFirstStart.Add(dummyDuration - dummyCFReqDur),
	})
	assert.DeepEqual(t, *mbpsSamples[2], Sample[float64]{
		Value:     400,
		Timestamp: dummySecondStart.Add(dummyIOEventsAfter[2]),
	})
	assert.DeepEqual(t, *mbpsSamples[3], Sample[float64]{
		Value:     480,
		Timestamp: dummySecondStart.Add(dummyDuration - dummyCFReqDur),
	})

	assert.Equal(t, sizeSum, 2*dummyMeasurementSize)
	assert.Equal(t, time.Duration(durationSum)*time.Microsecond, 2*dummyDuration)
}

func TestAnalyseMeasurements_ULWithZeroPoint(t *testing.T) {
	dummyMeasurementSize := int64(54 * 1000 * 1000) // 432 MBit
	dummyIOSizes := []int{
		5 * 1000 * 1000,  // 20 Mbit, 400 Mbps
		25 * 1000 * 1000, // 220 Mbit, 440 Mbps
		24 * 1000 * 1000, // 192 MBit, 480 Mbps
	}

	dummyCFReqDur := 20 * time.Millisecond
	dummyDuration := 1000*time.Millisecond + dummyCFReqDur
	dummyIOEventsAfter := []time.Duration{
		0,
		100 * time.Millisecond,
		600 * time.Millisecond,
	}

	dummyFirstStart := time.Now()
	dummySecondStart := dummyFirstStart.Add(2 * time.Second)

	dummyMeasurements := []*SpeedMeasurement{
		{
			Direction: DirectionUplink,
			Size:      dummyMeasurementSize,
			Start:     dummyFirstStart,
			End:       dummyFirstStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeRead, dummyFirstStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
		{
			Direction: DirectionUplink,
			Size:      dummyMeasurementSize,
			Start:     dummySecondStart,
			End:       dummySecondStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeRead, dummySecondStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
	}

	mbpsSamples, sizeSum, durationSum := analyseMeasurements(dummyMeasurements, true)

	assert.Equal(t, len(mbpsSamples), 6)
	assert.DeepEqual(t, *mbpsSamples[0], Sample[float64]{
		Value:     0,
		Timestamp: dummyFirstStart,
	})
	assert.DeepEqual(t, *mbpsSamples[1], Sample[float64]{
		Value:     400,
		Timestamp: dummyFirstStart.Add(dummyIOEventsAfter[2]),
	})
	assert.DeepEqual(t, *mbpsSamples[2], Sample[float64]{
		Value:     480,
		Timestamp: dummyFirstStart.Add(dummyDuration - dummyCFReqDur),
	})
	assert.DeepEqual(t, *mbpsSamples[3], Sample[float64]{
		Value:     0,
		Timestamp: dummySecondStart,
	})
	assert.DeepEqual(t, *mbpsSamples[4], Sample[float64]{
		Value:     400,
		Timestamp: dummySecondStart.Add(dummyIOEventsAfter[2]),
	})
	assert.DeepEqual(t, *mbpsSamples[5], Sample[float64]{
		Value:     480,
		Timestamp: dummySecondStart.Add(dummyDuration - dummyCFReqDur),
	})

	assert.Equal(t, sizeSum, 2*dummyMeasurementSize)
	assert.Equal(t, time.Duration(durationSum)*time.Microsecond, 2*dummyDuration)
}

func TestAnalyseMeasurements_ULWithCoincidentEvents(t *testing.T) {
	dummyMeasurementSize := int64(50 * 1000 * 1000) // 400 MBit
	dummyIOSizes := []int{
		20 * 1000 * 1000, // 160 Mbit
		30 * 1000 * 1000, // 240 MBit
		0,
	}

	dummyCFReqDur := 20 * time.Millisecond
	dummyDuration := 1000*time.Millisecond + dummyCFReqDur
	dummyIOEventsAfter := []time.Duration{
		0,
		0,
		dummyDuration - dummyCFReqDur,
	}

	dummyStart := time.Now()

	dummyMeasurements := []*SpeedMeasurement{
		{
			Direction: DirectionUplink,
			Size:      dummyMeasurementSize,
			Start:     dummyStart,
			End:       dummyStart.Add(dummyDuration),
			Duration:  dummyDuration,
			IOSampler: IOSampler{
				SizeWritten: dummyMeasurementSize,
				Events:      generateDummyIOEvents(IOModeRead, dummyStart, dummyIOEventsAfter, dummyIOSizes),
			},
			CFReqDur: dummyCFReqDur,
		},
	}

	mbpsSamples, sizeSum, durationSum := analyseMeasurements(dummyMeasurements, false)

	assert.Equal(t, len(mbpsSamples), 1)
	assert.DeepEqual(t, *mbpsSamples[0], Sample[float64]{
		Value:     400,
		Timestamp: dummyStart.Add(dummyDuration - dummyCFReqDur),
	})

	assert.Equal(t, sizeSum, dummyMeasurementSize)
	assert.Equal(t, time.Duration(durationSum)*time.Microsecond, dummyDuration)
}

func TestAnalyseMeasurementGroups(t *testing.T) {
	dummyMeasurementSize := int64(50 * 1000 * 1000) // 400 MBit
	dummyIOSizes := []int{
		5 * 1000 * 1000,  // 40 Mbit, 200 Mbps
		15 * 1000 * 1000, // 120 Mbit, 400 Mbps
		30 * 1000 * 1000, // 240 MBit, 480 Mbps
	}

	dummyCFReqDur := 20 * time.Millisecond
	dummyDuration := 1000*time.Millisecond + dummyCFReqDur
	dummyIOEventsAfter := []time.Duration{
		200*time.Millisecond + dummyCFReqDur,
		500*time.Millisecond + dummyCFReqDur,
		dummyDuration,
	}

	dummyStartBase := time.Now()
	dummyStartDriftMS := 10
	dummyStartTimestamps := []time.Time{
		dummyStartBase.Add(-time.Duration(dummyStartDriftMS) * time.Millisecond),
		dummyStartBase,
		dummyStartBase.Add(time.Duration(dummyStartDriftMS) * time.Millisecond),
	}

	dummyMeasurementGroups := [][]*SpeedMeasurement{
		{
			{
				Direction: DirectionDownlink,
				Size:      dummyMeasurementSize,
				Start:     dummyStartTimestamps[0],
				End:       dummyStartTimestamps[0].Add(dummyDuration),
				Duration:  dummyDuration,
				IOSampler: IOSampler{
					SizeWritten: dummyMeasurementSize,
					Events:      generateDummyIOEvents(IOModeWrite, dummyStartTimestamps[0], dummyIOEventsAfter, dummyIOSizes),
				},
				CFReqDur: dummyCFReqDur,
			},
		},
		{
			{
				Direction: DirectionDownlink,
				Size:      dummyMeasurementSize,
				Start:     dummyStartTimestamps[1],
				End:       dummyStartTimestamps[1].Add(dummyDuration),
				Duration:  dummyDuration,
				IOSampler: IOSampler{
					SizeWritten: dummyMeasurementSize,
					Events:      generateDummyIOEvents(IOModeWrite, dummyStartTimestamps[1], dummyIOEventsAfter, dummyIOSizes),
				},
				CFReqDur: dummyCFReqDur,
			},
		},
		{
			{
				Direction: DirectionDownlink,
				Size:      dummyMeasurementSize,
				Start:     dummyStartTimestamps[2],
				End:       dummyStartTimestamps[2].Add(dummyDuration),
				Duration:  dummyDuration,
				IOSampler: IOSampler{
					SizeWritten: dummyMeasurementSize,
					Events:      generateDummyIOEvents(IOModeWrite, dummyStartTimestamps[2], dummyIOEventsAfter, dummyIOSizes),
				},
				CFReqDur: dummyCFReqDur,
			},
		},
	}

	mbpsSamples, sizeSum, longestSpan := analyseMeasurementGroups(dummyMeasurementGroups)

	assert.Equal(t, len(mbpsSamples), 8)
	assert.DeepEqual(t, *mbpsSamples[0], Sample[float64]{
		Value:     320 + 0 + 0,
		Timestamp: dummyStartTimestamps[1].Add(dummyCFReqDur),
	})
	assert.DeepEqual(t, *mbpsSamples[1], Sample[float64]{
		Value:     320 + 320 + 0,
		Timestamp: dummyStartTimestamps[2].Add(dummyCFReqDur),
	})
	assert.DeepEqual(t, *mbpsSamples[2], Sample[float64]{
		Value:     320 + 320 + 320,
		Timestamp: dummyStartTimestamps[0].Add(dummyIOEventsAfter[1]),
	})
	assert.DeepEqual(t, *mbpsSamples[3], Sample[float64]{
		Value:     480 + 320 + 320,
		Timestamp: dummyStartTimestamps[1].Add(dummyIOEventsAfter[1]),
	})
	assert.DeepEqual(t, *mbpsSamples[4], Sample[float64]{
		Value:     480 + 480 + 320,
		Timestamp: dummyStartTimestamps[2].Add(dummyIOEventsAfter[1]),
	})
	assert.DeepEqual(t, *mbpsSamples[5], Sample[float64]{
		Value:     480 + 480 + 480,
		Timestamp: dummyStartTimestamps[0].Add(dummyIOEventsAfter[2]),
	})
	assert.DeepEqual(t, *mbpsSamples[6], Sample[float64]{
		Value:     0 + 480 + 480,
		Timestamp: dummyStartTimestamps[1].Add(dummyIOEventsAfter[2]),
	})
	assert.DeepEqual(t, *mbpsSamples[7], Sample[float64]{
		Value:     0 + 0 + 480,
		Timestamp: dummyStartTimestamps[2].Add(dummyIOEventsAfter[2]),
	})

	assert.Equal(t, sizeSum, 3*dummyMeasurementSize)
	assert.Equal(t, time.Duration(longestSpan)*time.Microsecond, (dummyDuration + time.Duration(2*dummyStartDriftMS)*time.Millisecond))
}

func TestAnalyseMeasurementGroups_WithCoincidentMeasurements(t *testing.T) {
	dummyMeasurementSize := int64(50 * 1000 * 1000) // 400 MBit
	dummyIOSizes := []int{
		0,
		0,
		50 * 1000 * 1000, // 400 Mbit, 400 Mbps
	}

	dummyCFReqDur := 20 * time.Millisecond
	dummyDuration := 1000*time.Millisecond + dummyCFReqDur
	dummyIOEventsAfter := []time.Duration{
		dummyCFReqDur,
		500*time.Millisecond + dummyCFReqDur,
		dummyDuration,
	}

	dummyCoincidentStart := time.Now()

	dummyCoincidentMeasurement := SpeedMeasurement{
		Direction: DirectionDownlink,
		Size:      dummyMeasurementSize,
		Start:     dummyCoincidentStart,
		End:       dummyCoincidentStart.Add(dummyDuration),
		Duration:  dummyDuration,
		IOSampler: IOSampler{
			SizeWritten: dummyMeasurementSize,
			Events:      generateDummyIOEvents(IOModeWrite, dummyCoincidentStart, dummyIOEventsAfter, dummyIOSizes),
		},
		CFReqDur: dummyCFReqDur,
	}

	dummyMeasurementGroups := [][]*SpeedMeasurement{{&dummyCoincidentMeasurement}, {&dummyCoincidentMeasurement}}

	mbpsSamples, sizeSum, longestSpan := analyseMeasurementGroups(dummyMeasurementGroups)

	assert.Equal(t, len(mbpsSamples), 1)
	assert.DeepEqual(t, *mbpsSamples[0], Sample[float64]{
		Value:     2 * 400,
		Timestamp: dummyCoincidentStart.Add(dummyIOEventsAfter[2]),
	})

	assert.Equal(t, sizeSum, 2*dummyMeasurementSize)
	assert.Equal(t, time.Duration(longestSpan)*time.Microsecond, dummyDuration)
}
