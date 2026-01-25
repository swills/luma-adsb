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

type IM struct {
	Advice          string `json:"advice"`
	BetaChangelog   string `json:"beta_changelog"`
	InChannelUpdate int    `json:"in_channel_update"`
	LatestDate      string `json:"latest_date"`
	LatestTag       string `json:"latest_tag"`
	MainChangelog   string `json:"main_changelog"`
	ShowUpdate      string `json:"show_update"`
}

func GetUpdateAvailable(ctx context.Context, host string, timeout time.Duration) (bool, error) {
	var err error

	var req *http.Request

	var res *http.Response

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statsDataURL := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/api/status/im",
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, statsDataURL.String(), nil)
	if err != nil {
		return false, fmt.Errorf("error creating http req: %w", err)
	}

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error making http request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("bad status making request: %w", err)
	}

	defer res.Body.Close()

	var body []byte

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return false, fmt.Errorf("error reading response body: %w", err)
	}

	var IMInfo IM

	err = json.Unmarshal(body, &IMInfo)
	if err != nil {
		return false, fmt.Errorf("unmarshal error parsing update status: %w", err)
	}

	if IMInfo.ShowUpdate == "1" {
		return true, nil
	}

	return false, nil
}
