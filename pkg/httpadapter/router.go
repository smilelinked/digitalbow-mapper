package httpadapter

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/smilelinkd/digitalbow-mapper/driver"
	"k8s.io/klog/v2"

	"github.com/smilelinkd/digitalbow-mapper/pkg/common"
	"github.com/smilelinkd/digitalbow-mapper/pkg/httpadapter/response"
)

// RestController the struct of HTTP route
type RestController struct {
	Router         *mux.Router
	reservedRoutes map[string]bool
	Client         *driver.DigitalbowClient
}

// NewRestController build a RestController
func NewRestController(r *mux.Router, dic *driver.DigitalbowClient) *RestController {
	return &RestController{
		Router:         r,
		reservedRoutes: make(map[string]bool),
		Client:         dic,
	}
}

// InitRestRoutes register the RESTful API
func (c *RestController) InitRestRoutes() {
	klog.V(1).Info("Registering v1 routes...")
	// common
	c.addReservedRoute(common.APIPingRoute, c.Ping).Methods(http.MethodGet)
	c.addReservedRoute(common.APIDeviceDownload, c.Download).Methods(http.MethodPost)
	c.addReservedRoute(common.APIDeviceExecute, c.Execute).Methods(http.MethodPost)
}

func (c *RestController) addReservedRoute(route string, handler func(http.ResponseWriter, *http.Request)) *mux.Route {
	c.reservedRoutes[route] = true
	return c.Router.HandleFunc(route, handler)
}

func (c *RestController) sendMapperError(
	writer http.ResponseWriter,
	request *http.Request,
	err string,
	API string) {
	correlationID := request.Header.Get(common.CorrelationHeader)
	if correlationID == "" {
		correlationID = "nil"
	}
	klog.Errorf("correlationID :%s error : %v", correlationID, err)
	c.sendResponse(writer, request, API, err, response.CodeMapping(common.KindServerError))
}

// sendResponse puts together the response packet for the V2 API
func (c *RestController) sendResponse(
	writer http.ResponseWriter,
	request *http.Request,
	API string,
	response interface{},
	statusCode int) {

	correlationID := request.Header.Get(common.CorrelationHeader)

	writer.Header().Set(common.CorrelationHeader, correlationID)
	writer.Header().Set(common.ContentType, common.ContentTypeJSON)
	writer.WriteHeader(statusCode)

	if response != nil {
		data, err := json.Marshal(response)
		if err != nil {
			klog.Error(fmt.Sprintf("Unable to marshal %s response", API), "error", err.Error(), common.CorrelationHeader, correlationID)
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = writer.Write(data)
		if err != nil {
			klog.Error(fmt.Sprintf("Unable to write %s response", API), "error", err.Error(), common.CorrelationHeader, correlationID)
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
