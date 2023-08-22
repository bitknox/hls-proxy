package main

import (
	"io"
	"net/http"

	parsing "github.com/bitknox/hls-proxy/parsing"
	"github.com/bitknox/hls-proxy/proxy"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

// Handler
func hello(c echo.Context) error {

	//parse incomming base64 query string and decde it into model struct
	input, err := parsing.ParseInputUrl(c.QueryParam("input"))

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

	//might not be needed
	for header, values := range resp.Header {
		c.Response().Header().Set(header, values[0])
	}

	defer resp.Body.Close()
	//add referer and origin headers if applicable

	io.Copy(c.Response().Writer, resp.Body)
	return nil
}
