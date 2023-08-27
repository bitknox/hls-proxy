package proxy

import (
	"io"
	"net/http"
	"time"

	"github.com/bitknox/hls-proxy/hls"
	"github.com/bitknox/hls-proxy/http_retry"
	"github.com/bitknox/hls-proxy/model"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// base useragent string
const USER_AGENT string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "

var preFetcher *hls.Prefetcher = hls.NewPrefetcherWithJanitor(40, 20*time.Second, 5*time.Hour, 30*time.Minute)

func ManifestProxy(c echo.Context, input *model.Input) error {

	req, err := http.NewRequest("GET", input.Url, nil)
	if err != nil {
		return err
	}

	addBaseHeaders(req, input)

	//send request to proxy
	resp, err := http_retry.ExecuteRetryableRequest(req, 3)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	//add referer and origin headers if applicable

	finalURL := resp.Request.URL
	//modify m3u8 file to point to proxy
	start := time.Now()
	bytes, err := io.ReadAll(resp.Body)
	res, err := hls.ModifyM3u8(string(bytes), finalURL, preFetcher)
	elapsed := time.Since(start)
	log.Debug("Modifying manifest took ", elapsed)
	c.Response().Status = http.StatusOK
	c.Response().Writer.Write([]byte(res))
	return nil
}

func TsProxy(c echo.Context, input *model.Input) error {
	//parse incomming base64 query string and decde it into model struct

	pId := c.QueryParam("pId")
	//check if we have the ts file in cache

	start := time.Now()
	if pId != "" {
		data, found := preFetcher.GetFetchedClip(pId, input.Url)
		if found {
			c.Response().Writer.Write(data)
			return nil
		}
	}
	elapsed := time.Since(start)

	log.Debug("Fetching clip from cache took ", elapsed)

	req, err := http.NewRequest("GET", input.Url, nil)

	addBaseHeaders(req, input)

	if err != nil {
		return err
	}

	//copy over range header if applicable
	if c.Request().Header.Get("Range") != "" {
		req.Header.Add("Range", c.Request().Header.Get("Range"))
	}

	log.Debug("Fetching clip from origin")

	//send request to original host
	resp, err := http_retry.ExecuteRetryableRequest(req, 3)

	//Some hls files have a content ranges for the same ts file
	//We therefore need to make sure that this is copied over to the response
	if resp.Header.Get("Content-Range") != "" {
		c.Response().Header().Set("Content-Range", resp.Header.Get("Content-Range"))
	}

	if resp.Header.Get("Content-Length") != "" {
		c.Response().Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	}

	defer resp.Body.Close()

	io.Copy(c.Response().Writer, resp.Body)
	return nil
}

func addBaseHeaders(req *http.Request, input *model.Input) {
	//add headers if applicable
	if input.Referer != "" {
		req.Header.Add("Referer", input.Referer)
	}
	if input.Origin != "" {
		req.Header.Add("Origin", input.Origin)
	}
	req.Header.Add("User-Agent", USER_AGENT)
}
