package adsb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Aircraft struct {
	Hex        string     `json:"hex"`
	MarkerType string     `json:"type"`
	CallSign   string     `json:"flight"`
	Latitude   float64    `json:"lat"`
	Longitude  float64    `json:"lon"`
	Altitude   json.Token `json:"alt_baro"`
	Category   string     `json:"category,omitempty"`
}

type Data struct {
	Planes []Aircraft `json:"aircraft"`
}

func GetADSBData(host string) (*Data, error) {
	var err error

	var req *http.Request

	var res *http.Response

	aircraftDataURL := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, "8080"),
		Path:   "/data/aircraft.json",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, aircraftDataURL.String(), nil)
	if err != nil {
		return &Data{}, fmt.Errorf("error creating http req: %w", err)
	}

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return &Data{}, fmt.Errorf("error making http request: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "bad status making request", "status", res.StatusCode)

		return &Data{}, fmt.Errorf("bad status making request: %w", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	var body []byte

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return &Data{}, fmt.Errorf("error reading response body: %w", err)
	}

	var myADSBData Data

	err = json.Unmarshal(body, &myADSBData)
	if err != nil {
		return &Data{}, fmt.Errorf("failed unmarshalling: %w", err)
	}

	return &myADSBData, nil
}
