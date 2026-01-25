package adsb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type CPUTempData struct {
	Age int    `json:"age"`
	CPU string `json:"cpu"`
}

func GetCPUTempC(ctx context.Context, host string, timeout time.Duration) (int, error) {
	var err error

	var req *http.Request

	var res *http.Response

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statsDataURL := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/api/get_temperatures.json",
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, statsDataURL.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("error creating http req: %w", err)
	}

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error making http request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad status making request: %w", err)
	}

	defer res.Body.Close()

	var body []byte

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	var CPUTempInfo CPUTempData

	err = json.Unmarshal(body, &CPUTempInfo)
	if err != nil {
		return 0, fmt.Errorf("unmarshal error parsing CPU Temp data: %w", err)
	}

	tempC, err := strconv.ParseInt(CPUTempInfo.CPU, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing CPU Temp: %w", err)
	}

	return int(tempC), nil
}
