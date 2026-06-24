package febbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (

	consoleMediaMovie = 1
	consoleMediaTV = 2

)

var (

	playBtnFIDRe = regexp.MustCompile(`(?is)<div[^>]*\bclass\s*=\s*"[^"]*\bplay_btn\b[^"]*"[^>]*\bdata-fid\s*=\s*"(\d+)"`)
	episodeInfoRe = regexp.MustCompile(`(?is)<div[^>]*\bclass\s*=\s*"[^"]*\bepisode_info\b[^"]*"[^>]*>`)
	episodeFileRe = regexp.MustCompile(`(?is)<div[^>]*\bclass\s*=\s*"[^"]*\bepisode_file\b[^"]*"[^>]*>`)

)

type consoleResponse struct {

	Code int `json:"code"`
	HTML string `json:"html"`

	Msg string `json:"msg"`

}

// GetMoviePlayFID resolves the Febbox file id bound to a movie via the console IMDb catalogue.
func (c *Client) GetMoviePlayFID(imdbID string) (int, error) {

	html, err := c.fetchConsoleHTML("/console/get_imdb_info2", url.Values{

		"imdb_id": {imdbID},
		"type": {strconv.Itoa(consoleMediaMovie)},

	})

	if err != nil {

		return 0, err

	}

	fid, ok := parseMoviePlayFID(html)

	if !ok {

		return 0, nil

	}

	return fid, nil

}

// GetEpisodePlayFID resolves the Febbox file id bound to one TV episode via the console catalogue.
func (c *Client) GetEpisodePlayFID(imdbID string, season, episode int) (int, error) {

	html, err := c.fetchConsoleHTML("/console/get_episode_list", url.Values{

		"imdb_id": {imdbID},
		"season": {strconv.Itoa(season)},

	})

	if err != nil {

		return 0, err

	}

	fid, ok := parseEpisodePlayFID(html, season, episode)

	if !ok {

		return 0, nil

	}

	return fid, nil

}

// GetConsoleLinks resolves stream qualities for a console-bound file id.
func (c *Client) GetConsoleLinks(fid int) ([]Quality, error) {

	return c.GetLinks("", fid, "")

}

func (c *Client) fetchConsoleHTML(path string, form url.Values) (string, error) {

	if err := c.requireCookie(""); err != nil {

		return "", err

	}

	endpoint := baseURL + path

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))

	if err != nil {

		return "", err

	}

	for key, value := range c.consoleHeaders() {

		request.Header.Set(key, value)

	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	request.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")

	response, err := c.client.Do(request)

	if err != nil {

		return "", err

	}

	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {

		return "", fmt.Errorf("febbox: fetch %s: %s", endpoint, response.Status)

	}

	body, err := io.ReadAll(response.Body)

	if err != nil {

		return "", err

	}

	var payload consoleResponse

	if err := json.Unmarshal(body, &payload); err != nil {

		return "", err

	}

	if payload.Code != 1 {

		if payload.Msg != "" {

			return "", fmt.Errorf("febbox: %s", payload.Msg)

		}

		return "", nil

	}

	return payload.HTML, nil

}

func (c *Client) consoleHeaders() map[string]string {

	headers := map[string]string{

		"user-agent": browserUA,
		"accept-language": "en-US,en;q=0.9",
		"referer": baseURL + "/console",

	}

	if c.cookie != "" {

		headers["cookie"] = "ui=" + c.cookie

	}

	return headers

}

func parseMoviePlayFID(html string) (int, bool) {

	match := playBtnFIDRe.FindStringSubmatch(html)

	if len(match) < 2 {

		return 0, false

	}

	fid, err := strconv.Atoi(match[1])

	if err != nil || fid <= 0 {

		return 0, false

	}

	return fid, true

}

func parseEpisodePlayFID(html string, season, episode int) (int, bool) {

	matches := episodeInfoRe.FindAllStringSubmatchIndex(html, -1)

	for _, match := range matches {

		if len(match) < 2 {

			continue

		}

		openTag := html[match[0]:match[1]]
		block, ok := innerHTMLUntilCloseTag(html, match[1], "div")

		if !ok {

			continue

		}

		blockSeason, seasonOK := parseIntAttr(openTag, "data-season")

		blockEpisode, episodeOK := parseIntAttr(openTag, "data-episode")

		if !seasonOK || !episodeOK || blockSeason != season || blockEpisode != episode {

			continue

		}

		if fid, ok := parseEpisodeFileFID(block); ok {

			return fid, true

		}

	}

	return 0, false

}

func parseEpisodeFileFID(block string) (int, bool) {

	matches := episodeFileRe.FindAllStringSubmatchIndex(block, -1)

	for _, match := range matches {

		if len(match) < 2 {

			continue

		}

		openTag := block[match[0]:match[1]]

		if strings.Contains(openTag, "no_episode_file") {

			continue

		}

		if fid, ok := parseFIDAttr(openTag, "data-id"); ok {

			return fid, true

		}

		if fid, ok := parseFIDAttr(openTag, "data-fid"); ok {

			return fid, true

		}

	}

	return 0, false

}

func parseFIDAttr(tag, name string) (int, bool) {

	value := extractAttr(tag, name)

	if value == "" {

		return 0, false

	}

	fid, err := strconv.Atoi(value)

	if err != nil || fid <= 0 {

		return 0, false

	}

	return fid, true

}

func parseIntAttr(tag, name string) (int, bool) {

	value := extractAttr(tag, name)

	if value == "" {

		return 0, false

	}

	parsed, err := strconv.Atoi(value)

	if err != nil {

		return 0, false

	}

	return parsed, true

}