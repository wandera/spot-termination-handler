package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

func waitForTerminationEvent() chan interface{} {
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
