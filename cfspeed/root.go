package cfspeed

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"time"
)

func printMetadata(metadata *MeasurementMetadata, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while fetching metadata: %v\n", err)
		os.Exit(1)
	} else if metadata != nil {
		fmt.Printf("SrcIP: %s (AS%s)\n", metadata.SrcIP, metadata.SrcASN)
		fmt.Printf("SrcLocation: %s, %s\n", metadata.SrcCity, metadata.SrcCountry)
		fmt.Printf("DstColocation: %s\n", metadata.DstColo)
	}
}

func printRTTMeasurement(measurement *Stats, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "RTT - Error during measurement: %v\n", err)
		os.Exit(1)
	} else if measurement != nil {
		fmt.Printf("RTT-mean: %.3f ms\n", measurement.Mean)
		fmt.Printf("RTT-stderr: %.3f ms\n", measurement.StdErr)
		fmt.Printf("RTT-min: %.3f ms\n", measurement.Min)
		fmt.Printf("RTT-max: %.3f ms\n", measurement.Max)
		fmt.Printf("RTT-n: %d\n", measurement.NSamples)
	}
}

func printAdaptiveSpeedMeasurement(label string, measurement *SpeedMeasurementStats, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s - Error during measurement: %v\n", label, err)
		os.Exit(1)
	} else if measurement != nil {
		fmt.Printf("%s-mean: %.3f Mbps\n", label, measurement.Mean)
		fmt.Printf("%s-stderr: %.3f Mbps\n", label, measurement.StdErr)
		fmt.Printf("%s-min: %.3f Mbps\n", label, measurement.Min)
		fmt.Printf("%s-max: %.3f Mbps\n", label, measurement.Max)
		fmt.Printf("%s-tx: %.3f MiB\n", label, math.Round(float64(measurement.TXSize)/1024/1024))
		fmt.Printf("%s-n: %d\n", label, measurement.NSamples)
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

func RunAndPrint(transportProtocol string) {
	SetTransportProtocol(transportProtocol)

	measurementMetadata, err := GetMeasurementMetadata()
	printMetadata(measurementMetadata, err)
	fmt.Println()

	measurementRTT, err := MeasureRTT()
	printRTTMeasurement(measurementRTT, err)
	fmt.Println()

	measurementDown, err := MeasureSpeedAdaptive(MeasureDownlink)
	printAdaptiveSpeedMeasurement("Downlink", measurementDown, err)
	fmt.Println()

	measurementUp, err := MeasureSpeedAdaptive(MeasureUplink)
	printAdaptiveSpeedMeasurement("Uplink", measurementUp, err)
}
