package main

import (
	"os"
	"strings"

	parsing "github.com/bitknox/hls-proxy/parsing"
	proxy "github.com/bitknox/hls-proxy/proxy"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
)

func main() {
	godotenv.Load()
	// Echo instance
	e := echo.New()

	logLevel := os.Getenv("LEVEL")

	if logLevel == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Middleware
	e.Use(middleware.CORS())
	//e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/:input", handle_request)

	// Start server
	if logLevel == "DEBUG" {
		e.Logger.Fatal(e.Start("localhost:1323"))
	} else {
		e.Logger.Fatal(e.Start(":1323"))
	}

}

func handle_request(c echo.Context) error {
	input, e := parsing.ParseInputUrl(c.Param("input"))
	if e != nil {
		return e
	}

	if strings.HasSuffix(input.Url, "m3u8") {
		return proxy.ManifestProxy(c, input)
	} else {
		return proxy.TsProxy(c, input)
	}
}

// Handler
