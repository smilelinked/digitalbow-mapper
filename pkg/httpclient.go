package pkg

import (
	"net/http"
	"time"

	"k8s.io/klog/v2"

	"github.com/gorilla/mux"
	"github.com/smilelinkd/digitalbow-mapper/driver"
	"github.com/smilelinkd/digitalbow-mapper/pkg/httpadapter"
)

// HTTPClient is structure used to init HttpClient
type HTTPClient struct {
	IP             string
	Port           string
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	server         *http.Server
	restController *httpadapter.RestController
}

// NewHTTPClient initializes a new Http client instance
func NewHTTPClient(dClient *driver.DigitalbowClient) *HTTPClient {
	return &HTTPClient{
		IP:             "0.0.0.0",
		Port:           "6666",
		WriteTimeout:   10 * time.Second,
		ReadTimeout:    10 * time.Second,
		restController: httpadapter.NewRestController(mux.NewRouter(), dClient),
	}
}

// Init is a method to construct HTTP server
func (hc *HTTPClient) Init() error {
	hc.restController.InitRestRoutes()
	hc.server = &http.Server{
		Addr:         hc.IP + ":" + hc.Port,
		WriteTimeout: hc.WriteTimeout,
		ReadTimeout:  hc.ReadTimeout,
		Handler:      hc.restController.Router,
	}
	klog.V(1).Info("HttpServer Start......")
	go func() {
		_, err := hc.Receive()
		if err != nil {
			klog.Errorf("Http Receive error:%v", err)
		}
	}()
	return nil
}

// UnInit is a method to close http server
func (hc *HTTPClient) UnInit() {
	err := hc.server.Close()
	if err != nil {
		klog.Error("Http server close err:", err.Error())
		return
	}
}

// Send no messages need to be sent
func (hc *HTTPClient) Send(message interface{}) error {
	return nil
}

// Receive http server start listen
func (hc *HTTPClient) Receive() (interface{}, error) {
	err := hc.server.ListenAndServe()
	if err != nil {
		return nil, err
	}
	return "", nil
}
