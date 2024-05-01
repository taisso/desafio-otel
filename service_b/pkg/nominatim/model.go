package nominatim

import "context"

type INominatim interface {
	GetLocation(ctx context.Context, cep string) (*Nominatim, error)
}

type Address struct {
	Postcode      string `json:"postcode"`
	Suburb        string `json:"suburb"`
	CityDistrict  string `json:"city_district"`
	City          string `json:"city"`
	Municipality  string `json:"municipality"`
	County        string `json:"county"`
	StateDistrict string `json:"state_district"`
	State         string `json:"state"`
	ISO31662Lvl4  string `json:"ISO3166-2-lvl4"`
	Region        string `json:"region"`
	Country       string `json:"country"`
	CountryCode   string `json:"country_code"`
}

type Nominatim struct {
	PlaceID     int      `json:"place_id"`
	Licence     string   `json:"licence"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	Category    string   `json:"category"`
	Type        string   `json:"type"`
	PlaceRank   int      `json:"place_rank"`
	Importance  float64  `json:"importance"`
	Addresstype string   `json:"addresstype"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Boundingbox []string `json:"boundingbox"`
	Address     Address  `json:"address"`
}
