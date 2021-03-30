package terminate

import (
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// Variable for testing purposes.
var metadataURI = "http://169.254.169.254/latest/meta-data/spot/instance-action"

func WaitCh() chan interface{} {
	ret := make(chan interface{})
	go func() {
		for {
			resp, err := http.Get(metadataURI)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode == 200 {
				close(ret)
				return
			}
			time.Sleep(5 * time.Second)
		}
	}()
	return ret
}
