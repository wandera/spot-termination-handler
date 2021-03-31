package terminate

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWaitCh(t *testing.T) {
	tests := []struct {
		name         string
		responseFunc func(writer http.ResponseWriter, request *http.Request)
		shouldFinish bool
		timeout      time.Duration
	}{
		{
			name: "meta endpoint responded 200",
			responseFunc: func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(200)
			},
			shouldFinish: true,
			timeout:      1 * time.Second,
		},
		{
			name: "meta endpoint responded 500",
			responseFunc: func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(500)
			},
			shouldFinish: false,
			timeout:      1 * time.Second,
		},
		{
			name: "meta endpoint not responded",
			responseFunc: func(writer http.ResponseWriter, request *http.Request) {
				time.Sleep(120 * time.Second)
			},
			shouldFinish: false,
			timeout:      1 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tt.responseFunc))
			metadataURI = ts.URL

			done := make(chan bool)
			go func() {
				<-WaitCh()
				done <- true
			}()

			select {
			case <-time.After(tt.timeout):
				if tt.shouldFinish {
					t.Fatal("the routine should have finished")
				}
			case <-done:
				if !tt.shouldFinish {
					t.Fatal("the routine shouldn't finish")
				}
			}
		})
	}
}
