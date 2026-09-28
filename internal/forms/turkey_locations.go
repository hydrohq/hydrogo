package forms

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

// Data snapshot: talhaekrem/TurkiyeSehirVeIlcelerJSON, MIT license.
// https://github.com/talhaekrem/TurkiyeSehirVeIlcelerJSON
//
//go:embed turkey_locations.json
var turkeyLocationsJSON []byte

// TurkishLocationFields provides the built-in province and district fields.
// Callers may change labels or IDs while retaining the supplied option map.
func TurkishLocationFields() ([]Field, error) {
	var data struct {
		Cities []struct {
			ID   int    `json:"Id"`
			Name string `json:"name"`
		} `json:"cities"`
		Districts []struct {
			CityID int    `json:"cityId"`
			Name   string `json:"name"`
		} `json:"districts"`
	}
	if err := json.Unmarshal(turkeyLocationsJSON, &data); err != nil {
		return nil, fmt.Errorf("decode bundled Turkish location data: %w", err)
	}
	provinces := make([]string, 0, len(data.Cities))
	byID := make(map[int]string, len(data.Cities))
	for _, city := range data.Cities {
		provinces = append(provinces, city.Name)
		byID[city.ID] = city.Name
	}
	districtsByProvince := make(map[string][]string, len(data.Cities))
	allDistricts := make(map[string]struct{}, len(data.Districts))
	for _, district := range data.Districts {
		province, ok := byID[district.CityID]
		if !ok {
			return nil, fmt.Errorf("district %q has unknown province", district.Name)
		}
		districtsByProvince[province] = append(districtsByProvince[province], district.Name)
		allDistricts[district.Name] = struct{}{}
	}
	for province := range districtsByProvince {
		sort.Strings(districtsByProvince[province])
	}
	districtOptions := make([]string, 0, len(allDistricts))
	for district := range allDistricts {
		districtOptions = append(districtOptions, district)
	}
	sort.Strings(provinces)
	sort.Strings(districtOptions)
	return []Field{
		{ID: "province", Type: Select, Label: "İl", Required: true, Options: provinces},
		{ID: "district", Type: CascadingSelect, Label: "İlçe", Required: true, Options: districtOptions, DependsOn: "province", CascadeOptions: districtsByProvince},
	}, nil
}
