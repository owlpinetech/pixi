package pixi

import (
	"io"
	"net/url"
	"os"
	"strings"
)

// OpenFileOrHttp opens a file from a local path or an HTTP(S) URL. If the path is a URL,
// it opens a buffered HTTP stream to reduce the number of individual reads of the file
// from the network; otherwise, it opens a local file.
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
