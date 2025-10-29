package pixi

import (
	"io"
	"net/url"
	"os"
	"strings"
)

func OpenFileOrHttp(path string) (io.ReadSeekCloser, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		pixiUrl, err := url.Parse(path)
		if err != nil {
			return nil, err
		}
		return OpenBufferedHttp(pixiUrl, nil)
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		return file, nil
	}
}
