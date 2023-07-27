package httpadapter

import (
	"encoding/json"
	"math"
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

	if _, ok := c.Client.Movements[executeRequest.Segment]; !ok {
		c.sendMapperError(writer, request, "The segment does not exist, please download first!", common.APIDeviceExecute)
		return
	}

	c.Client.SetStatus(common.StatusExecucting)
	time.Sleep(100 * time.Microsecond)
	go func() {
		trackData := c.Client.Movements[executeRequest.Segment]
		clylen := make([]float32, 6)
		for _, item := range trackData.MatrixList {
			bowResult := getBowDataformat(item)
			time.Sleep(100 * time.Microsecond)
			c.Client.Client.Execute(bowResult, clylen)
		}
	}()
	c.Client.SetStatus(common.StatusReady)

	c.sendResponse(writer, request, common.APIDeviceExecute, response, http.StatusOK)
}

type Matrix3x3 struct {
	m11, m12, m13 float64
	m21, m22, m23 float64
	m31, m32, m33 float64
}

type EulerAngle struct {
	roll, pitch, yaw float64
}

func matrixToEuler(m Matrix3x3) EulerAngle {
	var euler EulerAngle

	// Roll (X-axis rotation)
	euler.roll = math.Atan2(m.m32, m.m33)

	// Pitch (Y-axis rotation)
	sinPitch := -m.m31
	cosPitch := math.Sqrt(m.m32*m.m32 + m.m33*m.m33)
	euler.pitch = math.Atan2(sinPitch, cosPitch)

	// Yaw (Z-axis rotation)
	sinYaw := m.m21 / math.Cos(euler.pitch)
	cosYaw := m.m11 / math.Cos(euler.pitch)
	euler.yaw = math.Atan2(sinYaw, cosYaw)

	// Convert to degrees
	euler.roll = euler.roll * 180.0 / math.Pi
	euler.pitch = euler.pitch * 180.0 / math.Pi
	euler.yaw = euler.yaw * 180.0 / math.Pi

	return euler
}

func getBowDataformat(trackData [4][4]float64) []float32 {
	result := make([]float32, 6)
	rotationMatrix := Matrix3x3{
		m11: trackData[0][0],
		m12: trackData[0][1],
		m13: trackData[0][2],
		m21: trackData[1][0],
		m22: trackData[1][1],
		m23: trackData[1][2],
		m31: trackData[2][0],
		m32: trackData[2][1],
		m33: trackData[2][2],
	}

	eulerAngle := matrixToEuler(rotationMatrix)
	result[0] = float32(eulerAngle.roll)
	result[1] = float32(eulerAngle.pitch)
	result[2] = float32(eulerAngle.yaw)
	result[3] = float32(trackData[0][3] / 100)
	result[4] = float32(trackData[1][3] / 100)
	result[4] = float32(trackData[2][3] / 100)
	return result
}
