package model

import (
	"time"

	"github.com/urfave/cli/v2"
)

var (
	Configuration Config
)

type Config struct {
	Prefetch          bool
	SegmentCount      int
	Throttle          int
	Attempts          int
	ClipRetention     time.Duration
	PlaylistRetention time.Duration
	JanitorInterval   time.Duration
	UseHttps          bool
	DecryptSegments   bool
	Host              string
	Port              string
}

func InitializeConfig(c *cli.Context) {
	Configuration = Config{
		Prefetch:          c.Bool("prefetch"),
		SegmentCount:      c.Int("segments"),
		Throttle:          c.Int("throttle"),
		Attempts:          c.Int("attempts"),
		ClipRetention:     c.Duration("clip-retention"),
		PlaylistRetention: c.Duration("playlist-retention"),
		JanitorInterval:   c.Duration("janitor-interval"),
		DecryptSegments:   c.Bool("decrypt"),
		UseHttps:          c.Bool("https"),
		Host:              c.String("host"),
		Port:              c.String("port"),
	}
}
