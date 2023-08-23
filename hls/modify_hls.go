package hls

import (
	"strings"

	"github.com/labstack/gommon/log"
)

func ModifyM3u8(m3u8 string) (string, error) {
	var newManifest = strings.Builder{}

	if strings.Contains(m3u8, "RESOLUTION=") {
		log.Info("Master manifest detected")
	}

	return m3u8, nil
}
