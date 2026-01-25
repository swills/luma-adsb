package adsb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type FeederInfo struct {
	Enabled     bool
	BeastStatus string
	MLATStatus  string
}

func GetAllFeederStatus(ctx context.Context, host string, timeout time.Duration, config *MicroConfig) (*map[string]FeederInfo, error) {
	newFeederStatusInfo := map[string]FeederInfo{
		"adsbfi": {
			Enabled: config.AdsbfiIsEnabled,
		},
		"adsbhub": {
			Enabled: config.AdsbhubIsEnabled,
		},
		"adsblol": {
			Enabled: config.AdsblolIsEnabled,
		},
		"adsbx": {
			Enabled: config.AdsbxIsEnabled,
		},
		"alive": {
			Enabled: config.AliveIsEnabled,
		},
		"avdelphi": {
			Enabled: config.AVDelphiIsEnabled,
		},
		"flightaware": {
			Enabled: config.FlightAwareIsEnabled,
		},
		"flightradar": {
			Enabled: config.FlightRadarIsEnabled,
		},
		"opensky": {
			Enabled: config.OpenSkyIsEnabled,
		},
		"planefinder": {
			Enabled: config.PlaneFinderIsEnabled,
		},
		"planespotters": {
			Enabled: config.PlaneSpottersIsEnabled,
		},
		"planewatch": {
			Enabled: config.PlaneWatchIsEnabled,
		},
		"radarbox": {
			Enabled: config.RadarBoxIsEnabled,
		},
		"tat": {
			Enabled: config.TATIsEnabled,
		},
	}

	for k, v := range newFeederStatusInfo {
		if v.Enabled {
			newFeederStatus, err := GetFeederStatus(ctx, host, timeout, k)
			if err != nil {
				return nil, fmt.Errorf("error getting feeder status: %w", err)
			}
			t := newFeederStatusInfo[k]
			t.BeastStatus = newFeederStatus.Beast
			t.MLATStatus = newFeederStatus.MLAT
			newFeederStatusInfo[k] = t
		}
	}

	return &newFeederStatusInfo, nil
}

type FeederStatusWrapper struct {
	Wrapper FeederStatus `json:"0,omitempty"`
}

type FeederStatus struct {
	Beast string `json:"beast"`
	MLAT  string `json:"mlat"`
}

func GetFeederStatus(ctx context.Context, host string, timeout time.Duration, feederName string) (FeederStatus, error) {
	var err error

	var req *http.Request

	var res *http.Response

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statsDataURL := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/api/status/" + feederName,
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, statsDataURL.String(), nil)
	if err != nil {
		return FeederStatus{}, fmt.Errorf("error creating http req: %w", err)
	}

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return FeederStatus{}, fmt.Errorf("error making http request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return FeederStatus{}, fmt.Errorf("bad status making request: %w", err)
	}

	defer res.Body.Close()

	var body []byte

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return FeederStatus{}, fmt.Errorf("error reading response body: %w", err)
	}

	var newFeederStatus FeederStatusWrapper

	err = json.Unmarshal(body, &newFeederStatus)
	if err != nil {
		return FeederStatus{}, fmt.Errorf("unmarshal error parsing feeder status: %w", err)
	}

	return newFeederStatus.Wrapper, nil
}
