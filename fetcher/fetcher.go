package fetcher

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ctx = context.Background()
)

// Fetcher defines a common interface for types that fetch pricing data from the HCloud API.
type Fetcher interface {
	// GetHourly returns the prometheus collector that collects pricing data for hourly expenses.
	GetHourly() *prometheus.GaugeVec
	// GetMonthly returns the prometheus collector that collects pricing data for monthly expenses.
	GetMonthly() *prometheus.GaugeVec
	// GetCurrent returns the prometheus collector that collects pricing data for the resource so far.
	GetCurrent() *prometheus.GaugeVec
	// GetCurrent returns the prometheus collector that collects pricing data for the resource so far.
	GetCurrentCounter() *prometheus.CounterVec
	// Run executes a new data fetching cycle and updates the prometheus exposed collectors.
	Run(*hcloud.Client) error
}

type baseFetcher struct {
	pricing        *PriceProvider
	hourly         *prometheus.GaugeVec
	monthly        *prometheus.GaugeVec
	current        *prometheus.GaugeVec
	currentCounter *prometheus.CounterVec
}

func (fetcher baseFetcher) GetHourly() *prometheus.GaugeVec {
	return fetcher.hourly
}

func (fetcher baseFetcher) GetMonthly() *prometheus.GaugeVec {
	return fetcher.monthly
}

func (fetcher baseFetcher) GetCurrent() *prometheus.GaugeVec {
	return fetcher.current
}

func (fetcher baseFetcher) GetCurrentCounter() *prometheus.CounterVec {
	return fetcher.currentCounter
}

func newBase(pricing *PriceProvider, resource string, additionalLabels ...string) *baseFetcher {
	labels := []string{"name"}
	labels = append(labels, additionalLabels...)

	hourlyGaugeOpts := prometheus.GaugeOpts{
		Namespace: "hcloud",
		Subsystem: "pricing",
		Name:      fmt.Sprintf("%s_hourly", resource),
		Help:      fmt.Sprintf("The cost of the resource %s per hour", resource),
	}
	monthlyGaugeOpts := prometheus.GaugeOpts{
		Namespace: "hcloud",
		Subsystem: "pricing",
		Name:      fmt.Sprintf("%s_monthly", resource),
		Help:      fmt.Sprintf("The cost of the resource %s per month", resource),
	}
	currentGaugeOpts := prometheus.GaugeOpts{
		Namespace: "hcloud",
		Subsystem: "pricing",
		Name:      fmt.Sprintf("%s_current", resource),
		Help:      fmt.Sprintf("The cost of the resource %s so far", resource),
	}

	currentCounterCounterOpts := prometheus.CounterOpts{
		Namespace: "hcloud",
		Subsystem: "pricing",
		Name:      fmt.Sprintf("%s_current_counter", resource),
		Help:      fmt.Sprintf("The cost of the resource %s so far", resource),
	}

	return &baseFetcher{
		pricing:        pricing,
		hourly:         prometheus.NewGaugeVec(hourlyGaugeOpts, labels),
		monthly:        prometheus.NewGaugeVec(monthlyGaugeOpts, labels),
		current:        prometheus.NewGaugeVec(currentGaugeOpts, labels),
		currentCounter: prometheus.NewCounterVec(currentCounterCounterOpts, labels),
	}
}

// Fetchers defines a type for a slice of fetchers that should be handled together.
type Fetchers []Fetcher

// RegisterCollectors registers all collectors of the contained fetchers into the passed registry.
func (fetchers Fetchers) RegisterCollectors(registry *prometheus.Registry) {
	for _, fetcher := range fetchers {
		registry.MustRegister(
			fetcher.GetHourly(),
			fetcher.GetMonthly(),
			fetcher.GetCurrent(),
			fetcher.GetCurrentCounter(),
		)
	}
}

// Run executes all contained fetchers and returns a single error, even when multiple failures occurred.
func (fetchers Fetchers) Run(client *hcloud.Client) error {
	errors := prometheus.MultiError{}
	for _, fetcher := range fetchers {
		fetcher.GetHourly().Reset()
		fetcher.GetMonthly().Reset()
		fetcher.GetCurrent().Reset()
		fetcher.GetCurrentCounter().Reset()

		if err := fetcher.Run(client); err != nil {
			errors.Append(err)
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// MustRun executes all contained fetchers and panics if any of them threw an error.
func (fetchers Fetchers) MustRun(client *hcloud.Client) {
	if err := fetchers.Run(client); err != nil {
		panic(err)
	}
}
