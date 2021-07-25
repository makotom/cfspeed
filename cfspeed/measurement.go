package cfspeed

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	downURLTemplate = "https://speed.cloudflare.com/__down?bytes=%d"
	upURLTemplate   = "https://speed.cloudflare.com/__up?bytes=%d"

	rttMeasurementSoftTimeout = 2 * time.Second // Test element starts unless exceeding this duration

	adaptiveMeasurementBytesMin      = int64(64 * 1024)         // 64 KiB
	adaptiveMeasurementBytesMax      = int64(256 * 1024 * 1024) // 256 MiB
	adaptiveMeasurementExpBase       = 2                        // 64 k, 128 k, 256 k, 512 k, 1 M, 2 M, 4 M, 8 M, 16 M, 32 M, 64 M, 128 M, 256 M
	adaptiveMeasurementTimeThreshold = 2 * time.Second
	adaptiveMeasurementCount         = 5
)

func flushHTTPResponse(resp *http.Response) (int64, error) {
	flushedSize, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		return 0, err
	}
	err = resp.Body.Close()
	if err != nil {
		return 0, err
	}

	return flushedSize, nil
}

func GetMeasurementMetadata() (*MeasurementMetadata, error) {
	resp, err := http.Get(fmt.Sprintf(downURLTemplate, 0))
	if err != nil {
		return nil, err
	}
	_, err = flushHTTPResponse(resp)
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
		srcIP:      resp.Header.Get("cf-meta-ip"),
		srcASN:     resp.Header.Get("cf-meta-asn"),
		srcCity:    srcCity,
		srcCountry: srcCountry,
		dstColo:    resp.Header.Get("cf-meta-colo"),
	}, nil
}

func MeasureRTT() (*RTTStats, error) {
	durations := []time.Duration{}

	for start := time.Now(); time.Since(start) < rttMeasurementSoftTimeout; {
		measurement, err := MeasureUplink(0)
		if err != nil {
			return nil, err
		}
		durations = append(durations, measurement.duration)
	}

	durationStats := getDurationStats(&durations)

	return &RTTStats{
		nSamples: len(durations),
		mean:     durationStats.mean,
		stderr:   durationStats.stderr,
	}, nil
}

func MeasureDownlink(size int64) (*SpeedMeasurement, error) {
	getURL := fmt.Sprintf(downURLTemplate, size)

	start := time.Now()

	resp, err := http.Get(getURL)
	if err != nil {
		return nil, err
	}
	downloadedSize, err := flushHTTPResponse(resp)
	if err != nil {
		return nil, err
	}

	end := time.Now()

	return &SpeedMeasurement{
		size:     downloadedSize,
		duration: end.Sub(start),
	}, nil
}

func MeasureUplink(size int64) (*SpeedMeasurement, error) {
	postURL := fmt.Sprintf(upURLTemplate, size)
	postBodyReader := bytes.NewReader(make([]byte, size))

	start := time.Now()

	resp, err := http.Post(postURL, "application/octet-stream", postBodyReader)
	if err != nil {
		return nil, err
	}

	end := time.Now()

	_, err = flushHTTPResponse(resp)
	if err != nil {
		return nil, err
	}

	return &SpeedMeasurement{
		size:     size,
		duration: end.Sub(start),
	}, nil
}

func MeasureSpeedAdaptive(measurementFunc func(size int64) (*SpeedMeasurement, error)) (*SpeedMeasurementStats, error) {
	measurements := []*SpeedMeasurement{}
	measurementBytes := adaptiveMeasurementBytesMin

	for len(measurements) < adaptiveMeasurementCount {
		measurement, err := measurementFunc(measurementBytes)
		if err != nil {
			return nil, err
		}

		if len(measurements) == 0 && measurement.duration < adaptiveMeasurementTimeThreshold && measurementBytes < adaptiveMeasurementBytesMax {
			measurements = []*SpeedMeasurement{}
			measurementBytes *= adaptiveMeasurementExpBase
		} else {
			measurements = append(measurements, measurement)
		}
	}

	stats := getSpeedMeasurementStats(&measurements)

	return &SpeedMeasurementStats{
		nSamples: len(measurements),
		txSize:   measurementBytes,
		mean:     stats.mean,
		stderr:   stats.stderr,
	}, nil
}
