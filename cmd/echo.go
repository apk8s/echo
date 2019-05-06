package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	Version        = "0.0.0"
	ipEnv          = getEnv("IP", "0.0.0.0")
	portEnv        = getEnv("PORT", "1025")
	metricsIpEnv   = getEnv("METRICS_IP", ipEnv)
	metricsPortEnv = getEnv("METRICS_PORT", "2112")
	nodeName       = getEnv("NODE_NAME", "")
	podName        = getEnv("POD_NAME", "")
	namespace      = getEnv("NAMESPACE", "")
)

// main
func main() {
	var (
		ip          = flag.String("ip", ipEnv, "Server IP address to bind to.")
		port        = flag.String("port", portEnv, "Server port.")
		metricsPort = flag.String("metricsPort", metricsPortEnv, "Metrics port.")
		metricsIp   = flag.String("metricsIP", metricsIpEnv, "Falls back to same IP as server.")
	)

	flag.Parse()

	zapCfg := zap.NewProductionConfig()
	zapCfg.DisableCaller = true
	zapCfg.DisableStacktrace = true

	logger, err := zapCfg.Build()
	if err != nil {
		fmt.Printf("Can not build logger: %s\n", err.Error())
		os.Exit(1)
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())

		logger.Info("Starting Ok Metrics Server. ("+*metricsIp+":"+*metricsPort+"/metrics)",
			zap.String("type", "metrics_startup"),
			zap.String("port", *metricsPort),
			zap.String("ip", *metricsIp),
		)

		err = http.ListenAndServe(*metricsIp+":"+*metricsPort, nil)
		if err != nil {
			logger.Fatal("Error Starting Echo Metrics Server", zap.Error(err))
			os.Exit(1)
		}
	}()

	// TCP request handler
	handleTCPRequest := func(conn net.Conn, message string) {
		remote := conn.RemoteAddr().String()
		logger.Info("TCP connection OPEN", zap.String("remote", remote))

		defer func() {
			err := conn.Close()
			if err != nil {
				logger.Warn("Error closing connection.", zap.Error(err))
			}
		}()

		defer logger.Info("TCP connection CLOSED", zap.String("remote", conn.RemoteAddr().String()))

		// write initial message
		_, err := conn.Write([]byte(message))
		if err != nil {
			logger.Warn("Error writing echo", zap.Error(err))
		}

		// echo
		for {
			buf := make([]byte, 1024)
			size, err := conn.Read(buf)
			if err != nil {
				logger.Warn("Error reading buffer.", zap.Error(err))
				return
			}
			data := buf[:size]

			logger.Info("Received data",
				zap.String("remote", remote),
				zap.ByteString("data", data),
			)

			_, err = conn.Write(data)
			if err != nil {
				logger.Warn("Write data error", zap.Error(err))
				return
			}
		}
	}

	logger.Info("Starting TCP Echo Server",
		zap.String("type", "startup"),
		zap.String("port", *port),
		zap.String("ip", *ip),
	)

	l, err := net.Listen("tcp", *ip+":"+*port)
	if err != nil {
		logger.Fatal("Unable to establish TCP listener", zap.Error(err))
		os.Exit(1)
	}

	defer func() {
		err = l.Close()
		if err != nil {
			logger.Error("Unable to close TCP listener", zap.Error(err))
			os.Exit(1)
		}
	}()

	welcome := fmt.Sprintf(
		"Welcome to the TCP echo service running on the [%s] node in the [%s] namespace.\n",
		nodeName,
		namespace,
	)

	details := fmt.Sprintf(
		"Served by Pod [%s] running apk8s/echo version [%s].\n",
		podName,
		Version,
	)

	// handle connection
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Fatal("Error accepting connection.", zap.Error(err))
		}

		// handle TCP request in a go routine
		// send an initial message
		go handleTCPRequest(conn, welcome+details)
	}

}

// getEnv gets an environment variable or sets a default if
// one does not exist.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}
