package client

import (
	"fmt"
	"time"

	"mediakit/internal/providers/febbox"
	"mediakit/internal/providers/introdb"
	"mediakit/internal/providers/showbox"
	"mediakit/internal/vod"
)

// Movie returns a chainable handle for a movie by Showbox id.

func (c *Client) Movie(id int) *vod.Movie {

	return vod.NewMovie(c, id)

}

// Show returns a chainable handle for a TV series by Showbox id.

func (c *Client) Show(id int) *vod.Show {

	return vod.NewShow(c, id)

}

func (c *Client) GetFebBoxID(id, boxType int) (string, error) {

	key := fmt.Sprintf("%d:%d", boxType, id)

	c.shareKeyMu.Lock()

	if entry, ok := c.shareKeys[key]; ok && time.Now().Before(entry.expiry) {

		shareKey := entry.key

		c.shareKeyMu.Unlock()

		return shareKey, nil

	}

	c.shareKeyMu.Unlock()

	result, err, _ := c.shareKeyGroup.Do(key, func() (any, error) {

		c.shareKeyMu.Lock()

		if entry, ok := c.shareKeys[key]; ok && time.Now().Before(entry.expiry) {

			shareKey := entry.key

			c.shareKeyMu.Unlock()

			return shareKey, nil

		}

		c.shareKeyMu.Unlock()

		shareKey, err := c.showbox.GetFebBoxID(id, showbox.BoxType(boxType))

		if err != nil {

			return "", err

		}

		if shareKey != "" {

			c.shareKeyMu.Lock()

			c.shareKeys[key] = shareKeyCacheEntry{key: shareKey, expiry: time.Now().Add(shareKeyTTL)}

			c.shareKeyMu.Unlock()

		}

		return shareKey, nil

	})

	if err != nil {

		return "", err

	}

	return result.(string), nil

}

func (c *Client) GetConsoleMovieFID(imdbID string) (int, error) {

	return c.febbox.GetMoviePlayFID(imdbID)

}

func (c *Client) GetConsoleLinks(fid int) ([]febbox.Quality, error) {

	return c.febbox.GetConsoleLinks(fid)

}

func (c *Client) ListFiles(shareKey string, parentID any, cookie string) ([]febbox.File, error) {

	return c.febbox.ListFiles(shareKey, parentID, cookie)

}

func (c *Client) GetLinks(shareKey string, fid any, cookie string) ([]febbox.Quality, error) {

	return c.febbox.GetLinks(shareKey, fid, cookie)

}

func (c *Client) GetDownloadURL(shareKey string, fid any, cookie string) (string, error) {

	return c.febbox.GetDownloadURL(shareKey, fid, cookie)

}

func (c *Client) GetIntro(query introdb.MediaQuery) (*introdb.MediaRecord, error) {

	return c.intro.GetMedia(query)

}
