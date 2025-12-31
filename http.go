package gopixi

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HttpReadSeeker struct {
	url    *url.URL
	client *http.Client
	ctx    context.Context
	header http.Header
	size   int64
	offset int64
}

func OpenHttp(url *url.URL, client *http.Client) (*HttpReadSeeker, error) {
	if client == nil {
		client = http.DefaultClient
	}

	// determine whether the resource is rangeable
	resp, err := client.Head(url.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unsuccessful http request: response code %d", resp.StatusCode)
	}

	if !strings.Contains(resp.Header.Get("Accept-Ranges"), "bytes") {
		return nil, fmt.Errorf("the resource does not support byte range requests")
	}

	return &HttpReadSeeker{
		url:    url,
		client: client,
		size:   resp.ContentLength,
	}, nil
}

func (h *HttpReadSeeker) WithContext(ctx context.Context) *HttpReadSeeker {
	return &HttpReadSeeker{
		url:    h.url,
		client: h.client,
		ctx:    ctx,
		header: h.header,
		size:   h.size,
		offset: h.offset,
	}
}

func (h *HttpReadSeeker) WithHeader(header http.Header) *HttpReadSeeker {
	return &HttpReadSeeker{
		url:    h.url,
		client: h.client,
		ctx:    h.ctx,
		header: header,
		size:   h.size,
		offset: h.offset,
	}
}

func (h *HttpReadSeeker) Read(p []byte) (n int, err error) {
	if h.offset >= h.size {
		return 0, io.EOF
	}

	req, err := http.NewRequest("GET", h.url.String(), nil)
	if err != nil {
		return 0, err
	}

	// copy some http request properties
	if h.ctx != nil {
		req = req.WithContext(h.ctx)
	}
	if h.header != nil {
		for key, values := range h.header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// set the range header to read from the current offset
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", h.offset, h.size-1))

	resp, err := h.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	n, err = resp.Body.Read(p)
	if n > 0 {
		h.offset += int64(n)
	}

	// eof of current response body is not necessarily (or even usually) the end of the entire resource
	if err != nil && errors.Is(err, io.EOF) && h.offset+int64(n) < h.size {
		return n, nil
	}
	return n, err
}

func (h *HttpReadSeeker) Seek(offset int64, whence int) (int64, error) {
	newOffset := offset
	switch whence {
	case io.SeekStart:
		// nothing to do here
	case io.SeekCurrent:
		newOffset += h.offset
	case io.SeekEnd:
		newOffset = h.size + offset
	default:
		panic(fmt.Sprintf("invalid whence value: %d", whence))
	}

	if newOffset < 0 || newOffset > h.size {
		return 0, fmt.Errorf("seek out of bounds: %d", newOffset)
	}
	h.offset = newOffset

	return newOffset, nil
}

type BufferedHttpReadSeeker struct {
	HttpReadSeeker
	buffer *bufio.Reader
}

func OpenBufferedHttp(url *url.URL, client *http.Client) (*BufferedHttpReadSeeker, error) {
	httpReader, err := OpenHttp(url, client)
	if err != nil {
		return nil, err
	}

	buffer := bufio.NewReader(httpReader)

	return &BufferedHttpReadSeeker{
		HttpReadSeeker: *httpReader,
		buffer:         buffer,
	}, nil
}

func (b *BufferedHttpReadSeeker) Read(p []byte) (n int, err error) {
	return b.buffer.Read(p)
}

func (b *BufferedHttpReadSeeker) Seek(offset int64, whence int) (int64, error) {
	// reset the buffer before seeking
	b.buffer.Reset(&b.HttpReadSeeker)

	// perform the seek operation
	newOffset, err := b.HttpReadSeeker.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	// reset the buffer to the new offset
	b.buffer.Reset(&b.HttpReadSeeker)

	return newOffset, nil
}

func (b *BufferedHttpReadSeeker) Close() error {
	return nil
}
