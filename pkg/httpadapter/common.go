package httpadapter

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/smilelinkd/digitalbow-mapper/configmap"
	"github.com/smilelinkd/digitalbow-mapper/pkg/common"
	"k8s.io/klog/v2"
)

// Ping handles the requests to /ping endpoint. Is used to test if the service is working
// It returns a response as specified by the V1 API swagger in openAPI/common
func (c *RestController) Ping(writer http.ResponseWriter, request *http.Request) {
	response := "This is API " + common.APIVersion + ". Now is " + time.Now().Format(time.UnixDate)
	c.sendResponse(writer, request, common.APIPingRoute, response, http.StatusOK)
}

func (c *RestController) Download(writer http.ResponseWriter, request *http.Request) {
	response := "This is API " + common.APIVersion + ". Now is " + time.Now().Format(time.UnixDate)
	var downResultRequest configmap.DownloadRequest
	err := json.NewDecoder(request.Body).Decode(&downResultRequest)
	if err != nil {
		klog.Error("Bad request, failed to decode JSON: ", err)
		c.sendMapperError(writer, request, err.Error(), common.APIDeviceDownload)
		return
	}

	err = c.Client.DownloadResult(downResultRequest.Path, downResultRequest.Segment)
	if err != nil {
		klog.Error("Can't download file into memory: ", err)
		c.sendMapperError(writer, request, err.Error(), common.APIDeviceDownload)
		return
	}
	c.sendResponse(writer, request, common.APIDeviceDownload, response, http.StatusOK)
}
