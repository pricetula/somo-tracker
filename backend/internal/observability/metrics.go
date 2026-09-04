package observability

import (
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// defaultRegistry is the process-global default registry. Using the default
// registry satisfies the requirement "registered with the default registry".
var defaultRegistry = prometheus.DefaultRegisterer

// lazyCollectors holds the registered Go and process collectors so they are
// instantiated exactly once across the lifetime of the process (including
// repeated test runs).
var (
	lazyCollectorsMu sync.Mutex
	lazyCollectors   []prometheus.Collector
	lazyOnce         sync.Once
)

func registerDefaultCollectors() {
	lazyOnce.Do(func() {
		collectors := []prometheus.Collector{
			prometheus.NewGoCollector(),
			prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		}
		for _, c := range collectors {
			if err := defaultRegistry.Register(c); err != nil {
				if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
					panic("failed to register metric collector: " + err.Error())
				}
			}
		}
		lazyCollectors = collectors
	})
}

// RegisterMetrics registers the /metrics endpoint on the Fiber app and
// adds default Go runtime and process metrics to the Prometheus default
// registry. It is safe to call repeatedly — collectors are lazily
// instantiated once using sync.Once, and AlreadyRegisteredError is handled.
func RegisterMetrics(app *fiber.App) {
	registerDefaultCollectors()

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}

// MetricsInvoke is an Uber Fx invoke that registers the /metrics endpoint
// and default Go/process collectors when the application starts.
func MetricsInvoke(app *fiber.App) {
	RegisterMetrics(app)
}
