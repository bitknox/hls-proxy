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
	Throttle          time.Duration
	Attempts          int
	ClipRetention     time.Duration
	PlaylistRetention time.Duration
	JanitorInterval   time.Duration
}

func InitializeConfig(c *cli.Context) {
	Configuration = Config{
		Prefetch:          c.Bool("prefetch"),
		SegmentCount:      c.Int("segments"),
		Throttle:          c.Duration("throttle"),
		Attempts:          c.Int("attempts"),
		ClipRetention:     c.Duration("clip-retention"),
		PlaylistRetention: c.Duration("playlist-retention"),
		JanitorInterval:   c.Duration("janitor-interval"),
	}
}
