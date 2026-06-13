package febbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const baseURL = "https://www.febbox.com"

const browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"

// Options tunes a Client instance.
type Options struct {
	Cookie string
}

// Client browses Febbox shares.
type Client struct {
	cookie string
	client *http.Client
}

// New builds a Client with optional overrides.
func New(options Options) *Client {
	return &Client{
		cookie: options.Cookie,
		client: &http.Client{},
	}
}

// ListFiles lists the entries of a shared folder.
func (c *Client) ListFiles(shareKey string, parentID any, cookie string) ([]File, error) {
	url := fmt.Sprintf("%s/file/file_share_list?share_key=%s&pwd=&parent_id=%v&is_html=0", baseURL, shareKey, parentID)

	var data struct {
		Data struct {
			FileList []File `json:"file_list"`
		} `json:"data"`
	}

	if err := c.fetchJSON(url, shareKey, cookie, &data); err != nil {
		return nil, err
	}

	return data.Data.FileList, nil
}

// GetLinks resolves download qualities for a video file. Requires cookie.
func (c *Client) GetLinks(shareKey string, fid any, cookie string) ([]Quality, error) {
	if err := c.requireCookie(cookie); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/console/video_quality_list?fid=%v", baseURL, fid)

	var data struct {
		HTML string `json:"html"`
	}

	if err := c.fetchJSON(url, shareKey, cookie, &data); err != nil {
		return nil, err
	}

	return parseQualities(data.HTML), nil
}

// GetDownloadURL resolves a direct download link for a shared file.
func (c *Client) GetDownloadURL(shareKey string, fid any, cookie string) (string, error) {
	if err := c.requireCookie(cookie); err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/file/file_download_info?share_key=%s&fid=%v", baseURL, shareKey, fid)
	var payload struct {
		Data struct {
			DownloadURL string `json:"download_url"`
		} `json:"data"`
	}

	if err := c.fetchJSON(endpoint, shareKey, cookie, &payload); err != nil {
		return "", err
	}
	if payload.Data.DownloadURL == "" {
		return "", fmt.Errorf("febbox: empty download url for fid %v", fid)
	}
	return payload.Data.DownloadURL, nil
}

func (c *Client) requireCookie(cookie string) error {
	auth := cookie
	if auth == "" {
		auth = c.cookie
	}
	if auth != "" {
		return nil
	}
	return fmt.Errorf("febbox: auth cookie required")
}

func (c *Client) headers(shareKey, cookie string) map[string]string {
	headers := map[string]string{
		"user-agent":      browserUA,
		"accept-language": "en-US,en;q=0.9",
	}

	auth := cookie
	if auth == "" {
		auth = c.cookie
	}
	if auth != "" {
		headers["cookie"] = "ui=" + auth
	}
	if shareKey != "" {
		headers["referer"] = baseURL + "/share/" + shareKey
	}

	return headers
}

func (c *Client) fetchJSON(url, shareKey, cookie string, dest any) error {
	backoff := 3 * time.Second
	var last error

	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		err := c.doFetchJSON(url, shareKey, cookie, dest)
		if err == nil {
			return nil
		}

		last = err
		if !isRetryableStatus(err.Error()) {
			return err
		}
	}

	return last
}

func (c *Client) doFetchJSON(url, shareKey, cookie string, dest any) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	for key, value := range c.headers(shareKey, cookie) {
		request.Header.Set(key, value)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("febbox: fetch %s: %s", url, response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, dest)
}

var (
	fileQualityOpenRe = regexp.MustCompile(`(?is)<([a-z][a-z0-9]*)[^>]*\bfile_quality\b[^>]*>`)
	attrValueRe       = regexp.MustCompile(`(?i)([a-z0-9_-]+)\s*=\s*"([^"]*)"`)
	speedSpanRe       = regexp.MustCompile(`(?is)<[^>]*\bclass\s*=\s*"[^"]*\bspeed\b[^"]*"[^>]*>.*?<span[^>]*>(.*?)</span>`)
	tagStripRe        = regexp.MustCompile(`(?is)<[^>]+>`)
)

func parseQualities(html string) []Quality {
	matches := fileQualityOpenRe.FindAllStringSubmatchIndex(html, -1)
	qualities := make([]Quality, 0, len(matches))

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		openTag := html[match[0]:match[1]]
		tagName := html[match[2]:match[3]]
		contentStart := match[1]

		block, ok := innerHTMLUntilCloseTag(html, contentStart, tagName)
		if !ok {
			continue
		}

		qualities = append(qualities, Quality{
			URL:     extractAttr(openTag, "data-url"),
			Quality: extractAttr(openTag, "data-quality"),
			Name:    extractClassText(block, "name"),
			Speed:   extractSpeed(block),
			Size:    extractClassText(block, "size"),
		})
	}

	return qualities
}

func innerHTMLUntilCloseTag(html string, contentStart int, tagName string) (string, bool) {
	closeTag := "</" + strings.ToLower(tagName) + ">"
	lower := strings.ToLower(html[contentStart:])
	end := strings.Index(lower, closeTag)
	if end < 0 {
		return "", false
	}
	return html[contentStart : contentStart+end], true
}

func extractAttr(tag, name string) string {
	for _, match := range attrValueRe.FindAllStringSubmatch(tag, -1) {
		if len(match) < 3 {
			continue
		}
		if strings.EqualFold(match[1], name) {
			return match[2]
		}
	}
	return ""
}

func extractClassText(block, className string) string {
	pattern := fmt.Sprintf(`(?is)<[^>]*\bclass\s*=\s*"[^"]*\b%s\b[^"]*"[^>]*>(.*?)</[^>]+>`, regexp.QuoteMeta(className))
	match := regexp.MustCompile(pattern).FindStringSubmatch(block)
	if len(match) < 3 {
		return ""
	}
	return strings.TrimSpace(stripTags(match[2]))
}

func extractSpeed(block string) string {
	match := speedSpanRe.FindStringSubmatch(block)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(stripTags(match[1]))
}

func stripTags(value string) string {
	return tagStripRe.ReplaceAllString(value, "")
}