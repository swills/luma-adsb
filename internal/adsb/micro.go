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

type MicroConfig struct {
	MfVersion       string `json:"mf_version,omitempty"`
	SiteName        string `json:"site_name,omitempty"`
	AdsbxFeederID   string `json:"adsbxfeederid,omitempty"`
	Lat             string `json:"lat,omitempty"`
	Lon             string `json:"lon,omitempty"`
	Alt             string `json:"alt,omitempty"`
	Tz              string `json:"tz,omitempty"`
	MaxRange        int    `json:"max_range,omitempty"`
	Uat978IsEnabled bool   `json:"uat978--is_enabled,omitempty"`
	Lng             string `json:"lng,omitempty"`
	MlatPrivacy     bool   `json:"mlat_privacy,omitempty"`
	RouteApi        bool   `json:"route_api,omitempty"`
	Uat978          bool   `json:"uat978,omitempty"`
	AliveIsEnabled  bool   `json:"alive--is_enabled,omitempty"`

	HeyWhatsThat   bool   `json:"heywhatsthat,omitempty"`
	HeyWhatsThatID string `json:"hey_whats_that_id,omitempty"`

	AdsblolUUID     string `json:"adsblol_uuid,omitempty"`
	AdsblolLink     string `json:"adsblol_link,omitempty"`
	OpenskyUser     string `json:"opensky--user,omitempty"`
	UltrafeederUUID string `json:"ultrafeeder_uuid,omitempty"`

	AdsbhubKey        string `json:"adsbhub--key,omitempty"`
	FlightawareKey    string `json:"flightaware--key,omitempty"`
	FlightradarKey    string `json:"flightradar--key,omitempty"`
	FlightradarUATKey string `json:"flightradar_uat--key,omitempty"`
	OpenskyKey        string `json:"opensky--key,omitempty"`
	PlanefinderKey    string `json:"planefinder--key,omitempty"`
	PlanewatchKey     string `json:"planewatch--key,omitempty"`
	RadarboxKey       string `json:"radarbox--key,omitempty"`
	RadarboxsnKey     string `json:"radarbox--snkey,omitempty"`
	RadarvirtuelKey   string `json:"radarvirtuel--key,omitempty"`
	SDRMapKey         string `json:"sdrmap--key,omitempty"`
	SDRMapUser        string `json:"sdrmap--user,omitempty"`
	TenNintyUKKey     string `json:"1090uk--key,omitempty"`

	AdsbfiIsEnabled        bool `json:"adsbfi--is_enabled,omitempty"`
	AdsbhubIsEnabled       bool `json:"adsbhub--is_enabled,omitempty"`
	AdsblolIsEnabled       bool `json:"adsblol--is_enabled,omitempty"`
	AdsbxIsEnabled         bool `json:"adsbx--is_enabled,omitempty"`
	AVDelphiIsEnabled      bool `json:"avdelphi--is_enabled,omitempty"`
	FlightAwareIsEnabled   bool `json:"flightaware--is_enabled,omitempty"`
	FlightRadarIsEnabled   bool `json:"flightradar--is_enabled,omitempty"`
	FlyItalyIsEnabled      bool `json:"flyitaly--is_enabled,omitempty"`
	HPRadarIsEnabled       bool `json:"hpradar--is_enabled,omitempty"`
	OpenSkyIsEnabled       bool `json:"opensky--is_enabled,omitempty"`
	PlaneFinderIsEnabled   bool `json:"planefinder--is_enabled,omitempty"`
	PlaneSpottersIsEnabled bool `json:"planespotters--is_enabled,omitempty"`
	PlaneWatchIsEnabled    bool `json:"planewatch--is_enabled,omitempty"`
	RadarBoxIsEnabled      bool `json:"radarbox--is_enabled,omitempty"`
	RadarVirtuelIsEnabled  bool `json:"radarvirtuel--is_enabled,omitempty"`
	SDRMapIsEnabled        bool `json:"sdrmap--is_enabled,omitempty"`
	TATIsEnabled           bool `json:"tat--is_enabled,omitempty"`
	TenNintyUKIsEnabled    bool `json:"1090uk--is_enabled,omitempty"`
}

func GetMicroConfig(ctx context.Context, host string, timeout time.Duration) (*MicroConfig, error) {
	var err error

	var req *http.Request

	var res *http.Response

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statsDataURL := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/api/micro_settings",
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

	var foundStage2Stats MicroConfig

	err = json.Unmarshal(body, &foundStage2Stats)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling: %w", err)
	}

	return &foundStage2Stats, nil
}
