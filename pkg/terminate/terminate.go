package terminate

import (
	"io"
	"net/http"
	"time"
)

const (
	errBackoff             = 1 * time.Second
	invalidResponseBackoff = 5 * time.Second
)

// Variable for testing purposes.
var metadataURI = "http://169.254.169.254/latest/meta-data/spot/instance-action"

func WaitCh() chan interface{} {
	ret := make(chan interface{})
	go func() {
		for {
			resp, err := http.Get(metadataURI)
			if err != nil {
				time.Sleep(errBackoff)
				continue
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode == 200 {
				close(ret)
				return
			}
			time.Sleep(invalidResponseBackoff)
		}
	}()
	return ret
}
