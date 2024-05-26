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

func printRTTMeasurement(printer *log.Logger, label string, measurement *Stats) {
	if measurement != nil {
		printer.Printf("%s-mean: %.3f ms\n", label, measurement.Mean)
		printer.Printf("%s-stderr: %.3f ms\n", label, measurement.StdErr)
		printer.Printf("%s-min: %.3f ms\n", label, measurement.Min)
		printer.Printf("%s-max: %.3f ms\n", label, measurement.Max)
		printer.Printf("%s-deciles: %s ms\n", label, formatDeciles(measurement.Deciles))
		printer.Printf("%s-n: %d\n", label, measurement.NSamples)
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

func runAndPrintMeasurementMetadata(printer *log.Logger) error {
	measurementMetadata, err := GetMeasurementMetadata()

	if err != nil {
		return errors.Wrap(err, "could not fetch metadata")
	}

	printMetadata(printer, measurementMetadata)

	return nil
}

func runAndPrintUnloadedRTTMeasurement(printer *log.Logger) error {
	rttStats, _, err := MeasureRTT()

	if err != nil {
		return errors.Wrap(err, "RTT measurement failed")
	}

	printRTTMeasurement(printer, "RTT-Unloaded", rttStats)

	return nil
}

func runAndPrintDownlinkMeasurement(printer *log.Logger, multiplicity int, measureLoadedRTT bool) error {
	var dlStats *SpeedMeasurementStats
	var dlLoadedRTTStats *Stats
	var dlSpeedError error
	var dlLoadedRTTErr error

	dlLoadedRTTDone := make(chan bool)

	if measureLoadedRTT {
		go func() {
			time.Sleep(1000 * time.Millisecond)
			dlLoadedRTTStats, _, dlLoadedRTTErr = MeasureRTT()
			dlLoadedRTTDone <- true
		}()
	}

	if multiplicity > 0 {
		dlStats, dlSpeedError = MeasureDownlinkMultiplexed(multiplicity)
	} else {
		dlStats, dlSpeedError = MeasureDownlink()
	}
	if dlSpeedError != nil {
		return errors.Wrap(dlSpeedError, "downlink measurement failed")
	}

	printSpeedMeasurement(printer, "Downlink", dlStats)

	if measureLoadedRTT && <-dlLoadedRTTDone && dlLoadedRTTErr == nil {
		printer.Println()
		printRTTMeasurement(printer, "RTT-DownlinkLoaded", dlLoadedRTTStats)
	}

	return nil
}

func runAndPrintUplinkMeasurement(printer *log.Logger, multiplicity int, measureLoadedRTT bool) error {
	var ulStats *SpeedMeasurementStats
	var ulLoadedRTTStats *Stats
	var ulSpeedError error
	var ulLoadedRTTErr error

	ulLoadedRTTDone := make(chan bool)

	if measureLoadedRTT {
		go func() {
			time.Sleep(1000 * time.Millisecond)
			ulLoadedRTTStats, _, ulLoadedRTTErr = MeasureRTT()
			ulLoadedRTTDone <- true
		}()
	}

	if multiplicity > 0 {
		ulStats, ulSpeedError = MeasureUplinkMultiplexed(multiplicity)
	} else {
		ulStats, ulSpeedError = MeasureUplink()
	}
	if ulSpeedError != nil {
		return errors.Wrap(ulSpeedError, "uplink measurement failed")
	}

	printSpeedMeasurement(printer, "Uplink", ulStats)

	if measureLoadedRTT && <-ulLoadedRTTDone && ulLoadedRTTErr == nil {
		printer.Println()
		printRTTMeasurement(printer, "RTT-UplinkLoaded", ulLoadedRTTStats)
	}

	return nil
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

func RunAndPrint(printer *log.Logger, transportProtocol string, multiplicity int, measureRTT bool) error {
	SetTransportProtocol(transportProtocol, defaultDialTimeout)

	if err := runAndPrintMeasurementMetadata(printer); err != nil {
		return err
	}
	printer.Println()

	if measureRTT {
		if err := runAndPrintUnloadedRTTMeasurement(printer); err != nil {
			return err
		}
		printer.Println()
	}

	if err := runAndPrintDownlinkMeasurement(printer, multiplicity, measureRTT); err != nil {
		return err
	}
	printer.Println()

	if err := runAndPrintUplinkMeasurement(printer, multiplicity, measureRTT); err != nil {
		return err
	}

	return nil
}
