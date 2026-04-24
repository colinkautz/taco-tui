package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Store struct {
	Name        string
	StoreNumber string
}

type MenuItem struct {
	ProductID string
	Name      string
	Price     float64
}

type MenuCategory struct {
	ProductID string
	Name      string
	Items     []MenuItem
}

func (s Store) Title() string {
	return s.Name
}

func (s Store) Description() string {
	return fmt.Sprintf("Store %s", s.StoreNumber)
}

func (s Store) FilterValue() string {
	return s.Name
}

func fetchStores(lat, long float64) []Store {
	url := fmt.Sprintf("https://www.tacobell.com/tacobellwebservices/v4/tacobell/stores?latitude=%f&longitude=%f", lat, long)
	res, err := http.Get(url)

	if err != nil {
		return nil
	}
	defer res.Body.Close()

	var data struct {
		NearByStores []struct {
			StoreNumber string `json:"storeNumber"`
			Address     struct {
				Line1  string `json:"line1"`
				Town   string `json:"town"`
				Region struct {
					Isocode string `json:"isocode"`
				} `json:"region"`
			} `json:"address"`
		} `json:"nearByStores"`
	}

	if jsonErr := json.NewDecoder(res.Body).Decode(&data); jsonErr != nil {
		return nil
	}

	stores := make([]Store, 0, len(data.NearByStores))
	for _, store := range data.NearByStores {
		state := strings.TrimPrefix(store.Address.Region.Isocode, "US-")
		stores = append(stores, Store{
			Name:        fmt.Sprintf("%s, %s, %s", store.Address.Line1, store.Address.Town, state),
			StoreNumber: store.StoreNumber,
		})
	}
	return stores
}

func fetchMenu(storeNumber string) []MenuCategory {
	url := fmt.Sprintf("https://www.tacobell.com/tacobellwebservices/v4/tacobell/products/menu/%s", storeNumber)
	res, err := http.Get(url)

	if err != nil {
		return nil
	}
	defer res.Body.Close()

	var data struct {
		MenuProductCategories []struct {
			Code     string `json:"code"`
			Name     string `json:"name"`
			Products []struct {
				Code  string `json:"code"`
				Name  string `json:"name"`
				Price struct {
					Value float64 `json:"value"`
				} `json:"price"`
			} `json:"products"`
		} `json:"menuProductCategories"`
	}

	if jsonErr := json.NewDecoder(res.Body).Decode(&data); jsonErr != nil {
		return nil
	}

	categories := make([]MenuCategory, 0, len(data.MenuProductCategories))
	for _, category := range data.MenuProductCategories {
		var items []MenuItem

		for _, p := range category.Products {
			if p.Price.Value > 0 {
				items = append(items, MenuItem{
					ProductID: p.Code,
					Name:      p.Name,
					Price:     p.Price.Value,
				})
			}
		}

		if len(items) == 0 {
			continue
		}

		categories = append(categories, MenuCategory{
			ProductID: category.Code,
			Name:      category.Name,
			Items:     items,
		})
	}
	return categories
}

func lookupZip(zip string) (float64, float64) {
	if len(zip) < 5 {
		return 0, 0
	}

	url := fmt.Sprintf("https://niiknow.github.io/zipcode-us/db/%s/%s.json", zip[:2], zip[:5])
	resp, err := http.Get(url)

	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()

	var data struct {
		Lat  float64 `json:"lat"`
		Long float64 `json:"lng"`
	}

	if jsonErr := json.NewDecoder(resp.Body).Decode(&data); jsonErr != nil {
		return 0, 0
	}

	return data.Lat, data.Long
}
