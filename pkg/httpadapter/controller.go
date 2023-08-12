package httpadapter

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"github.com/smilelinkd/digitalbow-mapper/configmap"
	"github.com/smilelinkd/digitalbow-mapper/pkg/common"
	"gonum.org/v1/gonum/mat"
	"k8s.io/klog/v2"
)

// Ping handles the requests to /ping endpoint. Is used to test if the service is working
// It returns a response as specified by the V1 API swagger in openAPI/common
func (c *RestController) Ping(writer http.ResponseWriter, request *http.Request) {
	response := "This is API " + common.APIVersion + ". Now is " + time.Now().Format(time.UnixDate)
	c.sendResponse(writer, request, common.APIPingRoute, response, http.StatusOK)
}

func (c *RestController) Download(writer http.ResponseWriter, request *http.Request) {
	if c.Client.GetStatus() != common.StatusReady {
		c.sendMapperError(writer, request, "For now device is not ready please try next time!", common.APIDeviceDownload)
		return
	}
	response := "This is API " + common.APIVersion + ". Now is " + time.Now().Format(time.UnixDate)
	var downResultRequest configmap.DownloadRequest
	err := json.NewDecoder(request.Body).Decode(&downResultRequest)
	if err != nil {
		klog.Error("Bad request, failed to decode JSON: ", err)
		c.sendMapperError(writer, request, err.Error(), common.APIDeviceDownload)
		return
	}
	c.Client.SetStatus(common.StatusSyncing)
	err = c.Client.DownloadResult(downResultRequest.Path, downResultRequest.Segment)
	if err != nil {
		klog.Error("Can't download file into memory: ", err)
		c.sendMapperError(writer, request, err.Error(), common.APIDeviceDownload)
		c.Client.SetStatus(common.StatusReady)
		return
	}
	c.Client.SetStatus(common.StatusReady)
	c.sendResponse(writer, request, common.APIDeviceDownload, response, http.StatusOK)
}

func (c *RestController) Execute(writer http.ResponseWriter, request *http.Request) {
	if c.Client.GetStatus() != common.StatusReady {
		c.sendMapperError(writer, request, "For now device is not ready please try next time!", common.APIDeviceExecute)
		return
	}

	response := "This is API " + common.APIVersion + ". Now is " + time.Now().Format(time.UnixDate)
	var executeRequest configmap.ExecuteRequest
	err := json.NewDecoder(request.Body).Decode(&executeRequest)
	if err != nil {
		klog.Error("Bad request, failed to decode JSON: ", err)
		c.sendMapperError(writer, request, err.Error(), common.APIDeviceExecute)
		return
	}

	if _, ok := c.Client.Movements[executeRequest.Segment]; !ok && !executeRequest.Random {
		c.sendMapperError(writer, request, "The segment does not exist, please download first!", common.APIDeviceExecute)
		return
	}

	c.Client.SetStatus(common.StatusExecucting)

	options := serial.OpenOptions{
		PortName:        c.Client.Client.Config.SerialName,
		BaudRate:        uint(c.Client.Client.Config.BaudRate),
		DataBits:        uint(c.Client.Client.Config.DataBits),
		StopBits:        uint(c.Client.Client.Config.StopBits),
		ParityMode:      serial.PARITY_NONE,
		MinimumReadSize: 4,
	}

	go func() {
		port, err := serial.Open(options)
		if err != nil {
			klog.V(2).Infof("Error opening serial port... %v", err)
			return
		}
		defer port.Close()
		defer c.Client.SetStatus(common.StatusReady)

		if !executeRequest.Random {
			trackData := c.Client.Movements[executeRequest.Segment]
			aInit := make([]float64, 6)
			clylen := make([]float32, 6)
			for record, item := range trackData.MatrixList {
				bowResult := c.Client.GetBowDataformat(item, trackData.MatrixInit)
				if record == 0 {
					//# 记录第0帧的初始参数
					aInit = bowResult
				}
				matrixA := mat.NewDense(1, 6, bowResult)
				matrixInit := mat.NewDense(1, 6, aInit)
				var sixdofA mat.Dense
				sixdofA.Sub(matrixA, matrixInit)
				input64 := sixdofA.RawRowView(0)
				input32 := make([]float32, 6)
				for i := 0; i < 6; i++ {
					input32[i] = float32(input64[i])
				}
				c.Client.Client.Execute(input32, clylen)
				time.Sleep(100 * time.Millisecond)
				klog.V(2).Infof("execute with %v", clylen)
				_, err := port.Write(c.Client.AssembleSerialData(clylen))
				if err != nil {
					klog.Errorf("Error writing to serial port:%v ", err)
					return
				}
			}
		} else {
			clylen := make([]float32, 6)
			for i := 1; i <= 2; i++ {
				var bowResult []float32
				if len(executeRequest.Input) != 0 {
					if i%2 == 0 {
						bowResult = []float32{0, 0, 0, 0, 0, 0}
					} else {
						bowResult = executeRequest.Input
					}
				} else {
					bowResult = c.Client.RandomGetCylen(i)
				}
				c.Client.Client.Execute(bowResult, clylen)
				klog.V(2).Infof("execute output %v", clylen)
				writeMessage := c.Client.AssembleSerialData(clylen)
				hex_string_data := hex.EncodeToString(writeMessage)
				klog.V(2).Infof("serial output %s", hex_string_data)
				_, err := port.Write(writeMessage)
				if err != nil {
					klog.Errorf("Error writing to serial port:%v ", err)
					return
				}
				if executeRequest.Period != 0 {
					time.Sleep(time.Duration(executeRequest.Period) * time.Second)
				} else {
					time.Sleep(2 * time.Second)
				}
			}
		}
		// reset..
		_, err = port.Write(c.Client.ResetToZero())
		if err != nil {
			klog.Errorf("Error writing to serial port:%v ", err)
			return
		}
	}()

	c.sendResponse(writer, request, common.APIDeviceExecute, response, http.StatusOK)
}
