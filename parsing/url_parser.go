package parsing

import (
	"encoding/base64"

	"strings"

	"github.com/bitknox/hls-proxy/model"
)

func ParseInputUrl(inputString string) (*model.Input, error) {
	var out = &model.Input{}
	decodedBytes, err := base64.StdEncoding.DecodeString(inputString)
	parts := strings.Split(string(decodedBytes), "|")
	parts[0] = strings.TrimSpace(parts[0])

	if len(parts) == 1 {
		out.Url = parts[0]
		return out, nil
	}

	if len(parts) == 2 {
		out.Url = parts[0]
		out.Referer = parts[1]
		return out, nil
	}

	if len(parts) == 3 {
		out.Url = parts[0]
		out.Referer = parts[1]
		out.Origin = parts[2]
		return out, nil
	}

	if err != nil {
		return out, nil
	}

	return out, nil
}
