package cfspeed

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

var logger = log.New(os.Stderr, "", 0)

func printMetadata(printer *log.Logger, metadata *MeasurementMetadata, err error) {
	if err != nil {
		logger.Printf("Error while fetching metadata: %v\n", err)
	} else if metadata != nil {
		printer.Printf("SrcIP: %s (AS%s)\n", metadata.SrcIP, metadata.SrcASN)
		printer.Printf("SrcLocation: %s, %s\n", metadata.SrcCity, metadata.SrcCountry)
		printer.Printf("DstColocation: %s\n", metadata.DstColo)
	}
}

func printRTTMeasurement(printer *log.Logger, measurement *Stats, err error) {
	if err != nil {
		logger.Printf("RTT - Error during measurement: %v\n", err)
	} else if measurement != nil {
		printer.Printf("RTT-mean: %.3f ms\n", measurement.Mean)
		printer.Printf("RTT-stderr: %.3f ms\n", measurement.StdErr)
		printer.Printf("RTT-min: %.3f ms\n", measurement.Min)
		printer.Printf("RTT-max: %.3f ms\n", measurement.Max)
		printer.Printf("RTT-n: %d\n", measurement.NSamples)
	}
}

func printAdaptiveSpeedMeasurement(printer *log.Logger, label string, measurement *SpeedMeasurementStats, err error) {
	if err != nil {
		logger.Printf("%s - Error during measurement: %v\n", label, err)
	} else if measurement != nil {
		printer.Printf("%s-mean: %.3f Mbps\n", label, measurement.Mean)
		printer.Printf("%s-stderr: %.3f Mbps\n", label, measurement.StdErr)
		printer.Printf("%s-min: %.3f Mbps\n", label, measurement.Min)
		printer.Printf("%s-max: %.3f Mbps\n", label, measurement.Max)
		printer.Printf("%s-cat: %.3f Mbps\n", label, measurement.CatSpeed)
		printer.Printf("%s-tx: %.3f MiB\n", label, float64(measurement.TXSize)/1024/1024)
		printer.Printf("%s-ntx: %d\n", label, measurement.NTX)
		printer.Printf("%s-n: %d\n", label, measurement.NSamples)
	}
}

func SetTransportProtocol(protocol string) {
	// cf (1). https://go.googlesource.com/go/+/refs/tags/go1.16.6/src/net/http/transport.go#42
	// cf (2). https://go.googlesource.com/go/+/refs/tags/go1.16.6/src/net/http/transport.go#130
	http.DefaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   10 * time.Second,
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

func RunAndPrint(printer *log.Logger, transportProtocol string) error {
	SetTransportProtocol(transportProtocol)

	measurementMetadata, err := GetMeasurementMetadata()
	printMetadata(printer, measurementMetadata, err)
	if err != nil {
		return err
	}
	printer.Println()

	rttStats, cfReqDurStats, err := MeasureRTT()
	printRTTMeasurement(printer, rttStats, err)
	if err != nil {
		return err
	}
	printer.Println()

	downlinkStats, err := MeasureSpeedAdaptive("down", cfReqDurStats)
	printAdaptiveSpeedMeasurement(printer, "Downlink", downlinkStats, err)
	if err != nil {
		return err
	}
	printer.Println()

	uplinkStats, err := MeasureSpeedAdaptive("up", cfReqDurStats)
	printAdaptiveSpeedMeasurement(printer, "Uplink", uplinkStats, err)
	if err != nil {
		return err
	}

	return nil
}
