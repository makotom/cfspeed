package cfspeed

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultDialTimeout = 10 * time.Second
)

func printMetadata(printer *log.Logger, metadata *MeasurementMetadata) {
	if metadata != nil {
		printer.Printf("SrcIP: %s (AS%s)\n", metadata.SrcIP, metadata.SrcASN)
		printer.Printf("SrcLocation: %s, %s\n", metadata.SrcCity, metadata.SrcCountry)
		printer.Printf("DstColocation: %s\n", metadata.DstColo)
	}
}

func formatDeciles(deciles []float64) string {
	numStrs := []string{}

	for _, decile := range deciles {
		numStrs = append(numStrs, fmt.Sprintf("%.3f", decile))
	}

	return fmt.Sprintf("%v", numStrs)
}

func printRTTMeasurement(printer *log.Logger, measurement *Stats) {
	if measurement != nil {
		printer.Printf("RTT-mean: %.3f ms\n", measurement.Mean)
		printer.Printf("RTT-stderr: %.3f ms\n", measurement.StdErr)
		printer.Printf("RTT-min: %.3f ms\n", measurement.Min)
		printer.Printf("RTT-max: %.3f ms\n", measurement.Max)
		printer.Printf("RTT-deciles: %s ms\n", formatDeciles(measurement.Deciles))
		printer.Printf("RTT-n: %d\n", measurement.NSamples)
	}
}

func printSpeedMeasurement(printer *log.Logger, label string, measurement *SpeedMeasurementStats) {
	if measurement != nil {
		printer.Printf("%s-mean: %.3f Mbps\n", label, measurement.Mean)
		printer.Printf("%s-stderr: %.3f Mbps\n", label, measurement.StdErr)
		printer.Printf("%s-min: %.3f Mbps\n", label, measurement.Min)
		printer.Printf("%s-max: %.3f Mbps\n", label, measurement.Max)
		printer.Printf("%s-deciles: %s Mbps\n", label, formatDeciles(measurement.Deciles))
		printer.Printf("%s-cat: %.3f Mbps\n", label, measurement.CatSpeed)
		printer.Printf("%s-tx: %.3f MiB\n", label, float64(measurement.TXSize)/1024/1024)
		printer.Printf("%s-mx: %d\n", label, measurement.Multiplicity)
		printer.Printf("%s-n: %d\n", label, measurement.NSamples)
	}
}

func SetTransportProtocol(protocol string, dialTimeout time.Duration) {
	// cf. https://go.googlesource.com/go/+/refs/tags/go1.22.1/src/net/http/transport.go#43
	// cf. https://go.googlesource.com/go/+/refs/tags/go1.22.1/src/net/http/transport.go#140
	http.DefaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext(ctx, protocol, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func RunAndPrint(printer *log.Logger, transportProtocol string, multiplicity int) error {
	SetTransportProtocol(transportProtocol, defaultDialTimeout)

	measurementMetadata, err := GetMeasurementMetadata()
	if err != nil {
		return errors.Wrap(err, "could not fetch metadata")
	}
	printMetadata(printer, measurementMetadata)
	printer.Println()

	rttStats, _, err := MeasureRTT()
	if err != nil {
		return errors.Wrap(err, "RTT measurement failed")
	}
	printRTTMeasurement(printer, rttStats)
	printer.Println()

	var downlinkStats, uplinkStats *SpeedMeasurementStats

	if multiplicity > 0 {
		downlinkStats, err = MeasureDownlinkMultiplexed(multiplicity)
	} else {
		downlinkStats, err = MeasureDownlink()
	}
	if err != nil {
		return errors.Wrap(err, "downlink measurement failed")
	}
	printSpeedMeasurement(printer, "Downlink", downlinkStats)
	printer.Println()

	if multiplicity > 0 {
		uplinkStats, err = MeasureUplinkMultiplexed(multiplicity)
	} else {
		uplinkStats, err = MeasureUplink()
	}
	if err != nil {
		return errors.Wrap(err, "uplink measurement failed")
	}
	printSpeedMeasurement(printer, "Uplink", uplinkStats)

	return nil
}
