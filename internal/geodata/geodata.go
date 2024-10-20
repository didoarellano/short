package geodata

import (
	"net"
	"os"

	"github.com/ipinfo/go/v2/ipinfo"
)

type GeoData = ipinfo.Core

type GeoDataFetcher interface {
	GetGeoData(ip net.IP) (GeoData, error)
}

type RealGeoDataFetcher struct{}
type MockGeoDataFetcher struct{}

func (r *RealGeoDataFetcher) GetGeoData(ip net.IP) (GeoData, error) {
	token := os.Getenv("IPINFO_TOKEN")
	ipinfoClient := ipinfo.NewClient(nil, nil, token)
	geo, _ := ipinfoClient.GetIPInfo(ip)
	return *geo, nil
}

func (m *MockGeoDataFetcher) GetGeoData(ip net.IP) (GeoData, error) {
	return GeoData{
		IP:       ip,
		City:     "MockCity",
		Region:   "MockRegion",
		Country:  "MockCountry",
		Location: "",
		Org:      "",
		Postal:   "",
		Timezone: "",
	}, nil
}
