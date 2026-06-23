package showbox

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"mediakit/internal/textutil"
)

func envOr(key, fallback string) string {

	if v := os.Getenv(key); v != "" {

		return v

	}

	return fallback

}

var showboxWebURL = envOr("SHOWBOX_WEB_URL", "https://www.showbox.media")

var showbox = struct {

	baseURL string
	appKey string

	iv string
	key string

}{

	baseURL: envOr("SHOWBOX_API_URL", "https://mbpapi.shegu.net/api/api_client/index/"),
	appKey: "moviebox",

	iv: "wEiphTn!",
	key: "123d6cedf626dy54233aa1w6",

}

var baseParams = struct {

	appVersion string
	lang string

	platform string
	channel string

	appID string

	version string
	medium string

}{

	appVersion: "11.5",
	lang: "en",

	platform: "android",
	channel: "Website",

	appID: "27",

	version: "129",
	medium: "Website",

}

const requestTTLSeconds = 60 * 60 * 12
const searchCacheTTL = 60 * time.Minute

type searchCacheEntry struct {

	results []SearchResult
	expires time.Time

}

// Options tunes a Client instance.
type Options struct {

	ChildMode string

}

// Client talks to the Showbox catalogue.
type Client struct {

	childMode string

	client *http.Client

	searchMu sync.RWMutex
	searchCache map[string]searchCacheEntry

}

// New builds a Client with optional overrides.
func New(options Options) *Client {

	childMode := options.ChildMode

	if childMode == "" {

		childMode = "0"

	}

	return &Client{

		client: &http.Client{

			Timeout: 10 * time.Second,

		},

		searchCache: make(map[string]searchCacheEntry),

		childMode: childMode,

	}

}

// DecodeText normalizes user-facing strings from Showbox API payloads.
func DecodeText(value string) string {

	return textutil.DecodeHTML(value)

}

// TopHot returns trending search keywords from Showbox.
func (c *Client) TopHot(mediaType MediaType, pageLimit int) ([]string, error) {

	if mediaType != MediaMovie && mediaType != MediaTV {

		mediaType = MediaMovie

	}

	if pageLimit == 0 {

		pageLimit = 25

	}

	var keywords []string

	err := c.request("Search_hot", map[string]any{

		"type": mediaType,
		"pagelimit": pageLimit,

	}, &keywords)

	return keywords, err

}

// TopLists returns curated ranking categories for movies or TV shows.
func (c *Client) TopLists(boxType BoxType) ([]TopList, error) {

	var lists []TopList

	err := c.request("Top_list", map[string]any{"box_type": boxType}, &lists)

	return lists, err

}

// TopListMovies returns titles from a curated movie ranking.
func (c *Client) TopListMovies(listID string, page, pageLimit int) ([]SearchResult, error) {

	if page == 0 {

		page = 1

	}

	if pageLimit == 0 {

		pageLimit = 20

	}

	var results []SearchResult

	err := c.request("Top_list_movie", map[string]any{

		"id": listID,

		"page": page,
		"pagelimit": pageLimit,

	}, &results)

	return results, err

}

// TopListTV returns titles from a curated TV ranking.
func (c *Client) TopListTV(listID string, page, pageLimit int) ([]SearchResult, error) {

	if page == 0 {

		page = 1

	}

	if pageLimit == 0 {

		pageLimit = 20

	}

	var results []SearchResult

	err := c.request("Top_list_tv", map[string]any{

		"id": listID,

		"page": page,
		"pagelimit": pageLimit,

	}, &results)

	return results, err

}

func searchCacheKey(query string, mediaType MediaType, page, pageLimit int) string {

	return fmt.Sprintf("%s|%s|%d|%d", strings.ToLower(strings.TrimSpace(query)), mediaType, page, pageLimit)

}

// Search queries the catalogue for movies and/or shows matching query.
func (c *Client) Search(query string, mediaType MediaType, page, pageLimit int) ([]SearchResult, error) {

	if mediaType == "" {

		mediaType = MediaAll

	}

	if page == 0 {

		page = 1

	}

	if pageLimit == 0 {

		pageLimit = 20

	}

	key := searchCacheKey(query, mediaType, page, pageLimit)

	c.searchMu.RLock()

	if entry, ok := c.searchCache[key]; ok && time.Now().Before(entry.expires) {

		c.searchMu.RUnlock()
		return append([]SearchResult(nil), entry.results...), nil

	}

	c.searchMu.RUnlock()

	var results []SearchResult

	err := c.request("Search5", map[string]any{

		"keyword": query,

		"type": mediaType,
		"page": page,

		"pagelimit": pageLimit,

	}, &results)

	if err != nil {

		return nil, err

	}

	c.searchMu.Lock()

	c.searchCache[key] = searchCacheEntry{

		results: append([]SearchResult(nil), results...),
		expires: time.Now().Add(searchCacheTTL),

	}

	c.searchMu.Unlock()

	return results, nil

}

// GetMovie fetches full details for a movie by its Showbox id.
func (c *Client) GetMovie(movieID int) (map[string]any, error) {

	var data map[string]any

	err := c.request("Movie_detail", map[string]any{"mid": movieID}, &data)

	return data, err

}

// GetShow fetches full details for a TV series by its Showbox id.
func (c *Client) GetShow(showID int) (map[string]any, error) {

	var data map[string]any

	err := c.request("TV_detail_v2", map[string]any{"tid": showID}, &data)

	return data, err

}

// GetEpisodeList fetches episode titles for a TV season.
func (c *Client) GetEpisodeList(showID, season int) (map[int]string, error) {

	var data map[string]any

	err := c.request("TV_detail_v2", map[string]any{"tid": showID, "season": season}, &data)

	if err != nil {

		return nil, err

	}

	episodes, _ := data["episode"].([]any)
	titles := make(map[int]string, len(episodes))

	for _, item := range episodes {

		ep, ok := item.(map[string]any)

		if !ok {

			continue

		}

		epSeason, _ := ep["season"].(float64)

		if int(epSeason) != season {

			continue

		}

		num, _ := ep["episode"].(float64)
		title := DecodeText(fmt.Sprint(ep["title"]))

		if num > 0 && title != "" && title != "<nil>" {

			titles[int(num)] = title

		}

	}

	return titles, nil

}

// GetFebBoxID resolves the Febbox share key for a title.
func (c *Client) GetFebBoxID(id int, boxType BoxType) (string, error) {

	endpoint := fmt.Sprintf("%s/index/share_link?id=%d&type=%d", strings.TrimRight(showboxWebURL, "/"), id, boxType)

	response, err := c.client.Get(endpoint)

	if err != nil {

		return "", err

	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {

		return "", err

	}

	var parsed struct {

		Data *struct {

			Link string `json:"link"`

		} `json:"data"`

	}

	if len(body) == 0 {

		return "", nil

	}

	if err := json.Unmarshal(body, &parsed); err != nil {

		return "", err

	}

	if parsed.Data == nil || parsed.Data.Link == "" {

		return "", nil

	}

	parts := strings.Split(parsed.Data.Link, "/")

	return parts[len(parts)-1], nil

}

func encrypt(payload string) (string, error) {

	key := []byte(showbox.key)
	iv := []byte(showbox.iv)

	block, err := des.NewTripleDESCipher(key)

	if err != nil {

		return "", err

	}

	padded := pkcs7Pad([]byte(payload), block.BlockSize())
	ciphertext := make([]byte, len(padded))

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)

	return base64.StdEncoding.EncodeToString(ciphertext), nil

}

func sign(encrypted string) string {

	hashedKey := md5Hex(showbox.appKey)

	return md5Hex(hashedKey + showbox.key + encrypted)

}

func (c *Client) request(module string, params map[string]any, dest any) error {

	requestData := map[string]any{

		"childmode": c.childMode,

		"APP_VERSION": baseParams.appVersion,

		"LANG": baseParams.lang,
		"PLATFORM": baseParams.platform,

		"CHANNEL": baseParams.channel,

		"APPID": baseParams.appID,
		"VERSION": baseParams.version,

		"MEDIUM": baseParams.medium,

		"expired_date": time.Now().Unix() + requestTTLSeconds,
		"module": module,

	}

	for key, value := range params {

		requestData[key] = value

	}

	payload, err := json.Marshal(requestData)

	if err != nil {

		return err

	}

	encrypted, err := encrypt(string(payload))

	if err != nil {

		return err

	}

	envelope, err := json.Marshal(map[string]string{

		"app_key": md5Hex(showbox.appKey),
		"verify": sign(encrypted),
		"encrypt_data": encrypted,

	})

	if err != nil {

		return err

	}

	form := url.Values{

		"data": {base64.StdEncoding.EncodeToString(envelope)},

		"appid": {baseParams.appID},
		"platform": {baseParams.platform},
		"medium": {baseParams.medium},

		"version": {baseParams.version},

	}

	body := form.Encode() + "&token" + randomHex(32)

	request, err := http.NewRequest(http.MethodPost, showbox.baseURL, strings.NewReader(body))

	if err != nil {

		return err

	}

	request.Header.Set("Platform", baseParams.platform)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "okhttp/3.2.0")

	response, err := c.client.Do(request)

	if err != nil {

		return err

	}

	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)

	if err != nil {

		return err

	}

	if response.StatusCode == http.StatusTooManyRequests {

		return fmt.Errorf("showbox rate limited (HTTP 429)")

	}

	if response.StatusCode >= 400 {

		return fmt.Errorf("showbox HTTP %d: %s", response.StatusCode, truncateBody(raw, 120))

	}

	var wrapper struct {

		Data json.RawMessage `json:"data"`

		Code int `json:"code"`
		Msg string `json:"msg"`

	}

	if err := json.Unmarshal(raw, &wrapper); err != nil {

		return fmt.Errorf("showbox invalid response: %s", truncateBody(raw, 120))

	}

	if !showboxSuccess(wrapper.Code, wrapper.Msg) {

		return fmt.Errorf("showbox error %d: %s", wrapper.Code, wrapper.Msg)

	}

	if dest == nil {

		return nil

	}

	if len(wrapper.Data) == 0 || string(wrapper.Data) == "null" {

		return nil

	}

	if err := json.Unmarshal(wrapper.Data, dest); err != nil {

		return fmt.Errorf("showbox data parse: %w", err)

	}

	return nil

}

func showboxSuccess(code int, msg string) bool {

	switch code {

		case 0, 1, 200:

			return true

	}

	switch strings.ToLower(strings.TrimSpace(msg)) {

		case "success", "ok", "":

			return true

	}

	return false

}

func truncateBody(raw []byte, max int) string {

	text := strings.TrimSpace(string(raw))

	if len(text) > max {

		return text[:max] + "..."

	}

	return text

}

func randomHex(length int) string {

	bytes := make([]byte, length/2)
	_, _ = rand.Read(bytes)

	return hex.EncodeToString(bytes)

}

func md5Hex(value string) string {

	sum := md5.Sum([]byte(value))

	return hex.EncodeToString(sum[:])

}

func pkcs7Pad(data []byte, blockSize int) []byte {

	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)

	return append(data, padText...)

}
