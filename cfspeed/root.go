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
		fmt.Printf("SrcIP: %s (AS%s)\n", metadata.srcIP, metadata.srcASN)
		fmt.Printf("SrcLocation: %s, %s\n", metadata.srcCity, metadata.srcCountry)
		fmt.Printf("DstColocation: %s\n", metadata.dstColo)
	}
}

func printRTTMeasurement(measurement *RTTStats, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "RTT - Error during measurement: %v\n", err)
		os.Exit(1)
	} else if measurement != nil {
		fmt.Printf("RTT-mean: %.3f ms\n", measurement.mean)
		fmt.Printf("RTT-stderr: %.3f ms\n", measurement.stderr)
		fmt.Printf("RTT-N: %d\n", measurement.nSamples)
	}
}

func printAdaptiveSpeedMeasurement(label string, measurement *SpeedMeasurementStats, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s - Error during measurement: %v\n", label, err)
		os.Exit(1)
	} else if measurement != nil {
		fmt.Printf("%s-mean: %.3f Mbps\n", label, measurement.mean)
		fmt.Printf("%s-stderr: %.3f Mbps\n", label, measurement.stderr)
		fmt.Printf("%s-tx: %.3f MiB\n", label, math.Round(float64(measurement.txSize)/1024/1024))
		fmt.Printf("%s-N: %d\n", label, measurement.nSamples)
	}
}

func SetTransportProtocol(protocol string) {
	// cf (1). https://go.googlesource.com/go/+/refs/tags/go1.16.6/src/net/http/transport.go#42
	// cf (2). https://go.googlesource.com/go/+/refs/tags/go1.16.6/src/net/http/transport.go#130
	http.DefaultClient.Transport = &http.Transport{
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
