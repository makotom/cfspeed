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
	downURLTemplate = "https://speed.cloudflare.com/__down?bytes=%d"
	upURLTemplate   = "https://speed.cloudflare.com/__up"

	rttMeasurementDuration   = 2 * time.Second   // Measurement resumes by sending another ping until exceeding this time duration
	speedMeasurementDuration = 10 * time.Second  // Download / Upload continues until exceeding this time duration
	downloadSizeMax          = 512 * 1024 * 1024 // Maximum size of data to be downloaded; 512 MiB
	uploadSizeMax            = 512 * 1024 * 1024 // Maximum size of data to be uploaded; 512 MiB
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
	NSamples int
	TXSize   int64
	Mean     float64
	StdErr   float64
	Min      float64
	Max      float64
	Deciles  []float64
	CatSpeed float64
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
		Direction:      "down",
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
		Direction:      "up",
		Size:           postBodyReader.SizeRead,
		Start:          start,
		End:            end,
		Duration:       end.Sub(start),
		IOSampler:      postBodyReader.IOSampler,
		CFReqDur:       getCFReqDur(&resp.Header),
		HTTPRespHeader: resp.Header,
	}, nil
}

func doMeasureSpeed(measurementFunc func(_ int64, _ time.Time) (*SpeedMeasurement, error), measurementSizeMax int64) (*SpeedMeasurementStats, error) {
	measurements := []*SpeedMeasurement{}

	for measureUntil := time.Now().Add(speedMeasurementDuration); time.Since(measureUntil) < 0; {
		measurement, err := measurementFunc(measurementSizeMax, measureUntil)
		if err != nil {
			break
		}
		measurements = append(measurements, measurement)
	}

	stats, totalSize, totalDuration := getSpeedMeasurementStats(measurements)

	return &SpeedMeasurementStats{
		NSamples: stats.NSamples,
		TXSize:   totalSize,
		Mean:     stats.Mean,
		StdErr:   stats.StdErr,
		Min:      stats.Min,
		Max:      stats.Max,
		Deciles:  stats.Deciles,
		CatSpeed: float64(8*totalSize) / float64(totalDuration),
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

	for measureUntil := time.Now().Add(rttMeasurementDuration); time.Since(measureUntil) < 0; {
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

	return getDurationStats(durations), getDurationStats(cfReqDurs), nil
}

func MeasureDownlink() (*SpeedMeasurementStats, error) {
	return doMeasureSpeed(doDownlinkMeasurement, downloadSizeMax)
}

func MeasureUplink() (*SpeedMeasurementStats, error) {
	return doMeasureSpeed(doUplinkMeasurement, uploadSizeMax)
}
