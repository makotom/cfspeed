package cfspeed

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const (
	DirectionDownlink = "down"
	DirectionUplink   = "up"

	downURLTemplate = "https://speed.cloudflare.com/__down?bytes=%d"
	upURLTemplate   = "https://speed.cloudflare.com/__up"

	rttMeasurementDurationMax = 2 * time.Second   // Maximum duration of RTT measurement
	rttMeasurementMax         = 20                // Maximum number of pings to be made for RTT measurement
	speedMeasurementDuration  = 10 * time.Second  // Download / Upload continues until exceeding this time duration
	downloadSizeMax           = 512 * 1024 * 1024 // Maximum size of data to be downloaded; 512 MiB
	uploadSizeMax             = 512 * 1024 * 1024 // Maximum size of data to be uploaded; 512 MiB
)

type MeasurementMetadata struct {
	SrcIP      string
	SrcASN     string
	SrcCity    string
	SrcCountry string
	DstColo    string
}

type SpeedMeasurement struct {
	Direction      string
	Size           int64
	Start          time.Time
	End            time.Time
	Duration       time.Duration
	IOSampler      IOSampler
	CFReqDur       time.Duration
	HTTPRespHeader http.Header
}

type SpeedMeasurementStats struct {
	NSamples     int
	TXSize       int64
	Multiplicity int
	Mean         float64
	StdErr       float64
	Min          float64
	Max          float64
	Deciles      []float64
	CatSpeed     float64
}

func flushHTTPResponse(resp *http.Response, maxSize int64, flushUntil time.Time) (int64, *IOSampler, error) {
	drain := InitSamplingReaderWriter(maxSize, flushUntil)

	flushedSize, err := io.Copy(drain, resp.Body)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, nil, err
	}

	err = resp.Body.Close()

	return flushedSize, &drain.IOSampler, err
}

func getCFReqDur(httpRespHeader *http.Header) time.Duration {
	cfReqDur := time.Duration(0)

	cfReqDurMatch := regexp.MustCompile(`cfRequestDuration;dur=([\d.]+)`).FindStringSubmatch(httpRespHeader.Get("Server-Timing"))
	if len(cfReqDurMatch) > 0 {
		cfReqDur, _ = time.ParseDuration(fmt.Sprintf("%sms", cfReqDurMatch[1]))
	}

	return cfReqDur
}

func doDownlinkMeasurement(maxSize int64, measureUntil time.Time) (*SpeedMeasurement, error) {
	getURL := fmt.Sprintf(downURLTemplate, maxSize)
	start := time.Now()

	resp, err := http.Get(getURL)
	if err != nil {
		return nil, err
	}
	downloadedSize, ioSampler, err := flushHTTPResponse(resp, maxSize, measureUntil)
	if err != nil {
		return nil, err
	}

	end := time.Now()

	return &SpeedMeasurement{
		Direction:      DirectionDownlink,
		Size:           downloadedSize,
		Start:          start,
		End:            end,
		Duration:       end.Sub(start),
		IOSampler:      *ioSampler,
		CFReqDur:       getCFReqDur(&resp.Header),
		HTTPRespHeader: resp.Header,
	}, nil
}

func doUplinkMeasurement(maxSize int64, measureUntil time.Time) (*SpeedMeasurement, error) {
	postURL := upURLTemplate
	postBodyReader := InitSamplingReaderWriter(maxSize, measureUntil)

	start := time.Now()

	resp, err := http.Post(postURL, "application/octet-stream", postBodyReader)
	if err != nil {
		return nil, err
	}

	end := time.Now()

	_, _, err = flushHTTPResponse(resp, 0, measureUntil)
	if err != nil {
		return nil, err
	}

	return &SpeedMeasurement{
		Direction:      DirectionUplink,
		Size:           postBodyReader.SizeRead,
		Start:          start,
		End:            end,
		Duration:       end.Sub(start),
		IOSampler:      postBodyReader.IOSampler,
		CFReqDur:       getCFReqDur(&resp.Header),
		HTTPRespHeader: resp.Header,
	}, nil
}

func doMeasureSpeed(measurementFunc func(_ int64, _ time.Time) (*SpeedMeasurement, error), txSizeMax int64) ([]*SpeedMeasurement, error) {
	measurements := []*SpeedMeasurement{}
	var err error = nil

	for measureUntil := time.Now().Add(speedMeasurementDuration); time.Since(measureUntil) < 0; {
		measurement, err := measurementFunc(txSizeMax, measureUntil)
		if err != nil {
			break
		}
		measurements = append(measurements, measurement)
	}

	return measurements, err
}

func measureSpeedSingle(measurementFunc func(_ int64, _ time.Time) (*SpeedMeasurement, error), txSizeMax int64) (*SpeedMeasurementStats, error) {
	measurements, err := doMeasureSpeed(measurementFunc, txSizeMax)
	stats, totalSize, totalDuration := getSingleSpeedMeasurementStats(measurements)

	return &SpeedMeasurementStats{
		NSamples:     stats.NSamples,
		TXSize:       totalSize,
		Multiplicity: 1,
		Mean:         stats.Mean,
		StdErr:       stats.StdErr,
		Min:          stats.Min,
		Max:          stats.Max,
		Deciles:      stats.Deciles,
		CatSpeed:     float64(8*totalSize) / float64(totalDuration),
	}, err
}

func measureSpeedMultiplexed(measurementFunc func(_ int64, _ time.Time) (*SpeedMeasurement, error), txSizeMax int64, multiplicity int) (*SpeedMeasurementStats, error) {
	groupedMeasurements := make([][]*SpeedMeasurement, multiplicity)
	groupsCompleted := 0
	chanCompleted := make(chan error)

	for iter := 0; iter < multiplicity; iter += 1 {
		group := iter
		go func() {
			measurements, err := doMeasureSpeed(measurementFunc, txSizeMax)
			groupedMeasurements[group] = measurements
			chanCompleted <- err
		}()
	}

	for ; groupsCompleted < multiplicity; groupsCompleted += 1 {
		if err := <-chanCompleted; err != nil {
			return nil, err
		}
	}

	stats, totalSize, longestSpan := getMultiplexedSpeedMeasurementStats(groupedMeasurements)

	return &SpeedMeasurementStats{
		NSamples:     stats.NSamples,
		TXSize:       totalSize,
		Multiplicity: multiplicity,
		Mean:         stats.Mean,
		StdErr:       stats.StdErr,
		Min:          stats.Min,
		Max:          stats.Max,
		Deciles:      stats.Deciles,
		CatSpeed:     float64(8*totalSize) / float64(longestSpan),
	}, nil
}

func GetMeasurementMetadata() (*MeasurementMetadata, error) {
	resp, err := http.Get(fmt.Sprintf(downURLTemplate, 0))
	if err != nil {
		return nil, err
	}
	_, _, err = flushHTTPResponse(resp, 0, time.Now())
	if err != nil {
		return nil, err
	}

	srcCity := resp.Header.Get("cf-meta-city")
	if srcCity == "" {
		srcCity = "N/A"
	}

	srcCountry := resp.Header.Get("cf-meta-country")
	if srcCity == "" {
		srcCity = "N/A"
	}

	return &MeasurementMetadata{
		SrcIP:      resp.Header.Get("cf-meta-ip"),
		SrcASN:     resp.Header.Get("cf-meta-asn"),
		SrcCity:    srcCity,
		SrcCountry: srcCountry,
		DstColo:    resp.Header.Get("cf-meta-colo"),
	}, nil
}

func MeasureRTT() (*Stats, *Stats, error) {
	durations := []time.Duration{}
	cfReqDurs := []time.Duration{}

	for measureUntil := time.Now().Add(rttMeasurementDurationMax); time.Since(measureUntil) < 0 && len(durations) < rttMeasurementMax; {
		measurement, err := doUplinkMeasurement(0, time.Now())
		if err != nil {
			return nil, nil, err
		}

		cfReqDur := getCFReqDur(&measurement.HTTPRespHeader)
		cfReqDurs = append(cfReqDurs, cfReqDur)

		adjustedDuration := measurement.Duration - cfReqDur
		if adjustedDuration < 0 {
			adjustedDuration = 0
		}

		durations = append(durations, adjustedDuration)
	}

	return getDurationMSStats(durations), getDurationMSStats(cfReqDurs), nil
}

func MeasureDownlink() (*SpeedMeasurementStats, error) {
	return measureSpeedSingle(doDownlinkMeasurement, downloadSizeMax)
}

func MeasureDownlinkMultiplexed(multiplicity int) (*SpeedMeasurementStats, error) {
	return measureSpeedMultiplexed(doDownlinkMeasurement, downloadSizeMax, multiplicity)
}

func MeasureUplink() (*SpeedMeasurementStats, error) {
	return measureSpeedSingle(doUplinkMeasurement, uploadSizeMax)
}

func MeasureUplinkMultiplexed(multiplicity int) (*SpeedMeasurementStats, error) {
	return measureSpeedMultiplexed(doUplinkMeasurement, uploadSizeMax, multiplicity)
}
