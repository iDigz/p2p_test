package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"code", "method"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response time for handler in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler", "method"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// handler function that writes a simple response with a smiley
func handler(w http.ResponseWriter, r *http.Request) {
	timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues("/", r.Method))
	defer timer.ObserveDuration()

	time.Sleep(2 * time.Second)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "😊")

	httpRequestsTotal.WithLabelValues(fmt.Sprintf("%d", http.StatusOK), r.Method).Inc()
}

func main() {
	// Set up the HTTP server and define the route
	http.HandleFunc("/", handler)

	// Expose the Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Start the server on port 8080
	fmt.Println("Server is running on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting the server:", err)
	}
}
