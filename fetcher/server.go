package fetcher

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

var _ Fetcher = &server{}

// NewServer creates a new fetcher that will collect pricing information on servers.
func NewServer(pricing *PriceProvider) Fetcher {
	return &server{newBase(pricing, "server", "location", "type")}
}

type server struct {
	*baseFetcher
}

func (server server) Run(client *hcloud.Client) error {
	servers, _, err := client.Server.List(ctx, hcloud.ServerListOpts{})
	if err != nil {
		return err
	}

	for _, s := range servers {
		location := s.Datacenter.Location
		created := s.Created

		pricing, err := findServerPricing(location, s.ServerType.Pricings)
		if err != nil {
			return err
		}

		parseToGauge(server.hourly.WithLabelValues(s.Name, location.Name, s.ServerType.Name), pricing.Hourly.Gross)
		parseToGauge(server.monthly.WithLabelValues(s.Name, location.Name, s.ServerType.Name), pricing.Monthly.Gross)
		// fmt.Println("pricing.Hourly.Gross", pricing.Hourly.Gross)
		// fmt.Println("pricing.Hourly.Net", pricing.Hourly.Net)
		// fmt.Println("s.ServerType.Name", s.ServerType.Name)
		// fmt.Println("location.Name", location.Name)
		// fmt.Println("s.Name", s.Name)
		currentCostsAsString := getCurrentCosts(created, pricing.Hourly.Net)
		parseToGauge(server.current.WithLabelValues(s.Name, location.Name, s.ServerType.Name), currentCostsAsString)
		parseToCounter(server.currentCounter.WithLabelValues(s.Name, location.Name, s.ServerType.Name), currentCostsAsString)
	}

	return nil
}

func getCurrentCosts(created time.Time, hourlyPrice string) string {
	// timestamp := time.Date(2022, 9, 28, 16, 0, 0, 0, time.Now().Location())
	currentTime := time.Now()
	// fmt.Println("currentTime", currentTime)
	// fmt.Println("timestamp", timestamp)
	diff := currentTime.Sub(created).Hours()
	// fmt.Println("diff", diff)
	rounded := math.Ceil(diff)
	// fmt.Println("rounded", rounded)

	floatHourlyCosts, _ := strconv.ParseFloat(hourlyPrice, 32)
	// fmt.Println("cleaned", floatHourlyCosts)
	// hourlyCost, err := strconv.Atof(cleaned)
	// if err != nil {
	// 	// ... handle error
	// 	panic(err)
	// }
	currentCosts := floatHourlyCosts * rounded
	// fmt.Println("currentCosts", currentCosts)
	currentCostsAsString := fmt.Sprintf("%f", currentCosts)
	// fmt.Println("currentCostsAsString", currentCostsAsString)

	return currentCostsAsString
}

func findServerPricing(location *hcloud.Location, pricings []hcloud.ServerTypeLocationPricing) (*hcloud.ServerTypeLocationPricing, error) {
	for _, pricing := range pricings {
		if pricing.Location.Name == location.Name {
			return &pricing, nil
		}
	}

	return nil, fmt.Errorf("no server pricing found for location %s", location.Name)
}
