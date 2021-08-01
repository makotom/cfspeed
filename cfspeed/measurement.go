package cfspeed

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const (
	downURLTemplate = "https://speed.cloudflare.com/__down?bytes=%d"
	upURLTemplate   = "https://speed.cloudflare.com/__up"

	rttMeasurementSoftTimeout = 2 * time.Second // Test element starts unless exceeding this duration

	adaptiveMeasurementBytesMin      = int64(64 * 1024)         // 64 KiB
	adaptiveMeasurementBytesMax      = int64(512 * 1024 * 1024) // 512 MiB
	adaptiveMeasurementExpBase       = 2                        // 64 k, 128 k, 256 k, 512 k, 1 M, 2 M, 4 M, 8 M, 16 M, 32 M, 64 M, 128 M, 256 M, 512 M
	adaptiveMeasurementTimeThreshold = 2 * time.Second
	adaptiveMeasurementCount         = 5
)

func flushHTTPResponse(resp *http.Response) (int64, *IOSampler, error) {
	flusher := InitWriteSampler()

	flushedSize, err := io.Copy(flusher, resp.Body)
	if err != nil {
		return 0, nil, err
	}
	err = resp.Body.Close()
	if err != nil {
		return 0, nil, err
	}

	return flushedSize, &flusher.IOSampler, nil
}

func getCFReqDur(httpRespHeader *http.Header) time.Duration {
	cfReqDur := time.Duration(0)

	cfReqDurMatch := regexp.MustCompile(`cfRequestDuration;dur=([\d.]+)`).FindStringSubmatch(httpRespHeader.Get("Server-Timing"))
	if len(cfReqDurMatch) > 0 {
		cfReqDur, _ = time.ParseDuration(fmt.Sprintf("%sms", cfReqDurMatch[1]))
	}

	return cfReqDur
}

func GetMeasurementMetadata() (*MeasurementMetadata, error) {
	resp, err := http.Get(fmt.Sprintf(downURLTemplate, 0))
	if err != nil {
		return nil, err
	}
	_, _, err = flushHTTPResponse(resp)
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

	for start := time.Now(); time.Since(start) < rttMeasurementSoftTimeout; {
		measurement, err := MeasureUplink(0)
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

func MeasureDownlink(size int64) (*SpeedMeasurement, error) {
	getURL := fmt.Sprintf(downURLTemplate, size)

	start := time.Now()

	resp, err := http.Get(getURL)
	if err != nil {
		return nil, err
	}
	downloadedSize, ioSampler, err := flushHTTPResponse(resp)
	if err != nil {
		return nil, err
	}

	end := time.Now()

	return &SpeedMeasurement{
		Size:           downloadedSize,
		Start:          start,
		End:            end,
		Duration:       end.Sub(start),
		IOSampler:      *ioSampler,
		HTTPRespHeader: resp.Header,
	}, nil
}

func MeasureUplink(size int64) (*SpeedMeasurement, error) {
	postURL := upURLTemplate
	postBodyReader := InitReadSampler(size)

	start := time.Now()

	resp, err := http.Post(postURL, "application/octet-stream", postBodyReader)
	if err != nil {
		return nil, err
	}

	end := time.Now()

	_, _, err = flushHTTPResponse(resp)
	if err != nil {
		return nil, err
	}

	return &SpeedMeasurement{
		Size:           size,
		Start:          start,
		End:            end,
		Duration:       end.Sub(start),
		IOSampler:      postBodyReader.IOSampler,
		HTTPRespHeader: resp.Header,
	}, nil
}

func MeasureSpeedAdaptive(mode string, cfReqDurStats *Stats) (*SpeedMeasurementStats, error) {
	measurements := []*SpeedMeasurement{}
	cfReqDurs := []time.Duration{}
	measurementBytes := adaptiveMeasurementBytesMin

	measurementFunc := func(size int64) (*SpeedMeasurement, error) { return nil, fmt.Errorf("unknown mode %q", mode) }
	switch mode {
	case "down":
		measurementFunc = MeasureDownlink
	case "up":
		measurementFunc = MeasureUplink
	}

	for len(measurements) < adaptiveMeasurementCount {
		measurement, err := measurementFunc(measurementBytes)
		if err != nil {
			return nil, err
		}

		if len(measurements) == 0 && measurement.Duration < adaptiveMeasurementTimeThreshold && measurementBytes < adaptiveMeasurementBytesMax && measurementBytes == measurement.Size {
			measurements = []*SpeedMeasurement{}
			measurementBytes *= adaptiveMeasurementExpBase
		} else {
			measurements = append(measurements, measurement)
			cfReqDurs = append(cfReqDurs, getCFReqDur(&measurement.HTTPRespHeader))
			measurementBytes = measurement.Size
		}
	}

	if mode == "down" {
		cfReqDurStats = getDurationStats(cfReqDurs)
	}

	catSpeed, stats := getSpeedMeasurementStats(measurements, cfReqDurStats)

	return &SpeedMeasurementStats{
		NSamples: stats.NSamples,
		TXSize:   measurementBytes,
		NTX:      len(measurements),
		Mean:     stats.Mean,
		StdErr:   stats.StdErr,
		Min:      stats.Min,
		Max:      stats.Max,
		CatSpeed: catSpeed,
	}, nil
}
