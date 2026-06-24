package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const providerUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36"

var httpClient = &http.Client{Timeout: 15 * time.Second}

func getText(url string, extraHeaders map[string]string) (string, error) {

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {

		return "", err

	}

	req.Header.Set("User-Agent", providerUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")

	for k, v := range extraHeaders {

		req.Header.Set(k, v)

	}

	resp, err := httpClient.Do(req)

	if err != nil {

		return "", err

	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		return "", fmt.Errorf("http %d from %s", resp.StatusCode, url)

	}

	b, err := io.ReadAll(resp.Body)

	return string(b), err

}

func getJSON(url string, extraHeaders map[string]string) (map[string]any, error) {

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {

		return nil, err

	}

	req.Header.Set("User-Agent", providerUA)
	req.Header.Set("Accept", "application/json, */*")

	for k, v := range extraHeaders {

		req.Header.Set(k, v)

	}

	resp, err := httpClient.Do(req)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		return nil, fmt.Errorf("http %d from %s", resp.StatusCode, url)

	}

	b, err := io.ReadAll(resp.Body)

	if err != nil {

		return nil, err

	}

	var out map[string]any

	if err := json.Unmarshal(b, &out); err != nil {

		return nil, err

	}

	return out, nil

}

func matchFirst(re *regexp.Regexp, text string) string {

	if m := re.FindStringSubmatch(text); len(m) > 1 {

		return m[1]

	}

	return ""

}
