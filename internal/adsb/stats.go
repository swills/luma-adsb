package adsb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var ErrFailedToGetStats = errors.New("failed to get stats")

type Stage2stats struct {
	Pps         float32 `json:"pps"`
	Mps         float32 `json:"mps"`
	Uptime      int     `json:"uptime"`
	Planes      int     `json:"planes"`
	TotalPlanes int     `json:"tplanes"`
}

func GetStage2Stats(ctx context.Context, host string, timeout time.Duration) (*Stage2stats, error) {
	var err error

	var req *http.Request

	var res *http.Response

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statsDataURL := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/api/stage2_stats",
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, statsDataURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http req: %w", err)
	}

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status making request: %w", err)
	}

	defer res.Body.Close()

	var body []byte

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var foundStage2Stats []Stage2stats

	err = json.Unmarshal(body, &foundStage2Stats)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling: %w", err)
	}

	if len(foundStage2Stats) < 1 {
		return nil, ErrFailedToGetStats
	}

	return &foundStage2Stats[0], nil
}
