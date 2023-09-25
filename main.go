package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bitknox/hls-proxy/model"
	parsing "github.com/bitknox/hls-proxy/parsing"
	proxy "github.com/bitknox/hls-proxy/proxy"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "prefetch",
				Usage: "prefetch ts files",
				Value: true,
			},
			&cli.IntFlag{
				Name:  "segments",
				Usage: "how many segments to prefetch",
				Value: 30,
			},
			&cli.IntFlag{
				Name:  "throttle",
				Usage: "how much to throttle prefetch requests (requests per second)",
				Value: 5,
			},
			&cli.DurationFlag{
				Name:  "janitor-interval",
				Usage: "how often should the janitor clean the cache",
				Value: 20 * time.Second,
			},
			&cli.IntFlag{
				Name:  "attempts",
				Usage: "how many times to retry a request for a ts file",
				Value: 10,
			},
			&cli.DurationFlag{
				Name:  "clip-retention",
				Usage: "how long to keep ts files in cache",
				Value: 30 * time.Minute,
			},
			&cli.DurationFlag{
				Name:  "playlist-retention",
				Usage: "how long to keep playlists in cache",
				Value: 5 * time.Hour,
			},
			&cli.StringFlag{

				Name:  "host",
				Usage: "hostname to attach to proxy url",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "port",
				Usage: "port to attach to proxy url",
				Value: "1323",
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "log level",
				Value: "PRODUCTION",
			},
		},
		Name:  "hls-proxy",
		Usage: "start hls proxy server",
		Action: func(c *cli.Context) error {
			model.InitializeConfig(c)
			proxy.InitPrefetcher(&model.Configuration)
			fmt.Printf("%v", model.Configuration)
			launch_server("", c.Int("port"), c.String("log-level"))
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func launch_server(host string, port int, logLevel string) {
	godotenv.Load()
	// Echo instance
	e := echo.New()

	if logLevel == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Middleware
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/:input", handle_request)

	// Start server

	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, port)))

}

func handle_request(c echo.Context) error {
	input, e := parsing.ParseInputUrl(c.Param("input"))
	if e != nil {
		return e
	}
	//TODO: Not all m3u8 files end with m3u8
	if strings.HasSuffix(input.Url, "m3u8") {
		return proxy.ManifestProxy(c, input)
	} else {
		return proxy.TsProxy(c, input)
	}
}

// Handler
