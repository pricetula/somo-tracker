package observability

import (
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var defaultRegistry = prometheus.DefaultRegisterer

var lazyOnce sync.Once

func registerDefaultCollectors() {
	lazyOnce.Do(func() {
		collectors := []prometheus.Collector{
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		}
		for _, c := range collectors {
			if err := defaultRegistry.Register(c); err != nil {
				if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
					panic("failed to register metric collector: " + err.Error())
				}
			}
		}
	})
}

func RegisterMetrics(app *fiber.App) {
	registerDefaultCollectors()

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}

func MetricsInvoke(app *fiber.App) {
	RegisterMetrics(app)
}
