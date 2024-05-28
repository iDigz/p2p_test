# Setting Up a Scalable Web Server with Prometheus Monitoring and Alertmanager

This guide will help you set up a Go web server that scales with load, is monitored by Prometheus, and sends alerts via Alertmanager.

## Step 1: Create a Go Web Server

Create a directory for your project and navigate into it. Then create a file named main.go with the following content:

```go
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

func handler(w http.ResponseWriter, r *http.Request) {
	timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues("/", r.Method))
	defer timer.ObserveDuration()

	time.Sleep(2 * time.Second)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "😊")

	httpRequestsTotal.WithLabelValues(fmt.Sprintf("%d", http.StatusOK), r.Method).Inc()
}

func main() {
	http.HandleFunc("/", handler)

	http.Handle("/metrics", promhttp.Handler())

	fmt.Println("Server is running on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting the server:", err)
	}
}

```

## Step 2: Create a Dockerfile

```dockerfile
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main .
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

## Step 3: Create a Docker Compose File

Create a docker-compose.yml file to define your services:

```yaml
version: '3.8'

services:
  web:
    build: .
    ports:
      - "8080"
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: "0.5"
          memory: "512M"
      restart_policy:
        condition: on-failure

  nginx:
    image: nginx:latest
    ports:
      - "8080:80"
    depends_on:
      - web
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
    depends_on:
      - web
      - alertmanager

  alertmanager:
    image: prom/alertmanager:latest
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml
    ports:
      - "9093:9093"
```

## Step 4: Configure NGINX for Load Balancing

Create an nginx.conf file to configure NGINX as a load balancer:

```nginx configuration
events {}

http {
    upstream web_servers {
        server web:8080;
    }

    server {
        listen 80;

        location / {
            proxy_pass http://web_servers;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /metrics {
            proxy_pass http://web_servers;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

## Step 5: Configure Prometheus

Create a prometheus.yml file to configure Prometheus:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'go-server'
    static_configs:
      - targets: ['web:8080']

rule_files:
  - 'alert_rules.yml'

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - 'alertmanager:9093'
```

## Step 6: Create Alert Rules

Create an alert_rules.yml file to define alerting rules:

```yaml
groups:
  - name: example
    rules:
      - alert: InstanceDown
        expr: up == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Instance {{ $labels.instance }} down"
          description: "{{ $labels.instance }} of job {{ $labels.job }} has been down for more than 5 minutes."

      - alert: HighResponseTime
        expr: http_request_duration_seconds_bucket{le="1"} / http_request_duration_seconds_count > 0.9
        for: 10s
        labels:
          severity: warning
        annotations:
          summary: "High response time on {{ $labels.instance }}"
          description: "Response time on {{ $labels.instance }} is higher than 1 second for more than 1 minute."
```

## Step 7: Configure Alertmanager

Create an alertmanager.yml file to configure Alertmanager:

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 3h
  receiver: 'webhook'

receivers:
  - name: 'webhook'
    webhook_configs:
      - url: 'http://localhost:9093/'
```

## Step 8: Build and Run the Services

Use Docker Compose to build and run the services:

`docker-compose up --build`

## Step 9: Verify the Setup

    Prometheus: Access Prometheus at http://localhost:9090 and ensure it is scraping the metrics.
    Alertmanager: Access Alertmanager at http://localhost:9093 and ensure it is configured correctly.
    Web Server: Access the web server through NGINX at http://localhost:8080.

## Testing Alerts

InstanceDown Alert: Stop one of the web server instances to trigger the InstanceDown alert:

`docker-compose stop web`

HighResponseTime Alert: Simulate high response time by modifying the handler function in main.go to introduce a delay.
Check the alerts in Prometheus and verify that Alertmanager sends the alerts to the configured webhook.