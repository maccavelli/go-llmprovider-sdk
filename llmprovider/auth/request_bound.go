package auth

import (
	"context"
	"io"
	"net/http"
	"time"
)

// authRequestTimeout bounds each auth request, from send until its body is
// closed. The default client has no total timeout, since a generation is
// bounded by its idle limit instead (0021-MADR D2), so an auth request carries
// its own bound. It is a variable so that tests can shorten it.
var authRequestTimeout = 30 * time.Second

// doBounded sends req under authRequestTimeout. The bound lasts until the
// reply's body is closed, so a caller may read the body after doBounded
// returns.
func doBounded(client *http.Client, req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), authRequestTimeout)
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

// cancelOnClose ends a request's context when its body is closed.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}
