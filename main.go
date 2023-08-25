package main

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	hls "github.com/bitknox/hls-proxy/hls"
	"github.com/bitknox/hls-proxy/model"
	parsing "github.com/bitknox/hls-proxy/parsing"
	"github.com/bitknox/hls-proxy/proxy"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	godotenv.Load()
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/:input", handle_request)

	// Start server
	go e.Logger.Fatal(e.Start("localhost:1323"))

}

func handle_request(c echo.Context) error {
	input, e := parsing.ParseInputUrl(c.Param("input"))
	if e != nil {
		return e
	}

	if strings.HasSuffix(input.Url, "m3u8") {
		return manifest_proxy(c, input)
	} else {
		return ts_proxy(c, input)
	}
}

// Handler
func manifest_proxy(c echo.Context, input *model.Input) error {

	req, err := http.NewRequest("GET", input.Url, nil)
	if err != nil {
		return err
	}

	//add headers if applicable
	if input.Referer != "" {
		req.Header.Add("Referer", input.Referer)
	}
	if input.Origin != "" {
		req.Header.Add("Origin", input.Origin)
	}
	req.Header.Add("User-Agent", proxy.USER_AGENT)

	//send request to proxy
	resp, err := proxy.Proxy.Client.Do(req)

	if err != nil {
		return err
	}

	//might not be needed
	/*for header, values := range resp.Header {
		c.Response().Header().Set(header, values[0])
	}*/

	defer resp.Body.Close()
	//add referer and origin headers if applicable

	finalURL := resp.Request.URL
	//modify m3u8 file to point to proxy
	start := time.Now()
	bytes, err := io.ReadAll(resp.Body)
	res, err := hls.ModifyM3u8(string(bytes), finalURL)
	elapsed := time.Since(start)
	log.Printf("Modifying manifest took %s", elapsed)
	c.Response().Writer.Write([]byte(res))
	c.Response().Status = http.StatusOK
	return nil
}

func ts_proxy(c echo.Context, input *model.Input) error {
	//parse incomming base64 query string and decde it into model struct

	req, err := http.NewRequest("GET", input.Url, nil)

	if err != nil {
		return err
	}

	//copy over range header if applicable
	if c.Request().Header.Get("Range") != "" {
		req.Header.Add("Range", c.Request().Header.Get("Range"))
	}

	//add headers if applicable
	if input.Referer != "" {
		req.Header.Add("Referer", input.Referer)
	}
	if input.Origin != "" {
		req.Header.Add("Origin", input.Origin)
	}
	req.Header.Add("User-Agent", proxy.USER_AGENT)

	//send request to original host
	resp, err := proxy.Proxy.Client.Do(req)

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
