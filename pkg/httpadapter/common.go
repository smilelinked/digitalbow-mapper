package httpadapter

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/jacobsa/go-serial/serial"
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

	if _, ok := c.Client.Movements[executeRequest.Segment]; !ok && !executeRequest.Random {
		c.sendMapperError(writer, request, "The segment does not exist, please download first!", common.APIDeviceExecute)
		return
	}

	c.Client.SetStatus(common.StatusExecucting)
	time.Sleep(100 * time.Microsecond)

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

		if !executeRequest.Random {
			trackData := c.Client.Movements[executeRequest.Segment]
			clylen := make([]float32, 6)
			for _, item := range trackData.MatrixList {
				bowResult := getBowDataformat(item)
				time.Sleep(100 * time.Microsecond)
				c.Client.Client.Execute(bowResult, clylen)
				klog.V(2).Infof("execute with %v", clylen)
				_, err := port.Write(assembleSerialData(clylen))
				if err != nil {
					klog.Errorf("Error writing to serial port:%v ", err)
					return
				}
			}
		} else {
			clylen := make([]float32, 6)
			for i := 1; i <= 10; i++ {
				time.Sleep(7 * time.Second)
				var bowResult []float32
				if len(executeRequest.Input) != 0 {
					if i%2 == 0 {
						bowResult = []float32{2, 0, 0, 0, 0, 0}
					}
					bowResult = executeRequest.Input
				} else {
					bowResult = randomGetCylen(i)
				}
				c.Client.Client.Execute(bowResult, clylen)
				klog.V(2).Infof("execute output %v", clylen)
				_, err := port.Write(assembleSerialData(clylen))
				if err != nil {
					klog.Errorf("Error writing to serial port:%v ", err)
					return
				}
			}
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

func assembleSerialData(moves []float32) []byte {
	b := []byte{0x55, 0xAA, 0x13, 0xFF, 0xF3}
	sum := 0x13 + 0xFF + 0xF3
	for i, item := range moves {
		id := byte(i)
		number1 := int32((item - 0.1569) * 40000)
		number1_2 := byte(number1)
		number1_1 := byte(number1 >> 8)
		b = append(b, id, number1_1, number1_2)
		sum += i + int(byte(number1)) + int(number1_2) + int(number1_1)
	}
	b = append(b, byte(sum))
	return b
}

func randomGetCylen(i int) []float32 {
	if i%3 == 0 {
		return []float32{2, 0, 0, 0, 0, 0}
	} else if i%3 == 1 {
		return []float32{-5, 5, -5, 0.01, -0.01, 0.01}
	} else {
		return []float32{5, -5, 5, -0.01, 0.01, -0.01}
	}
}
