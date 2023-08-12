/*
Copyright 2020 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

//#cgo CFLAGS: -I./number
//#cgo LDFLAGS: -L${SRCDIR}/number -lSixDof -lm
//
//#include <stdio.h>
//#include <stdlib.h>
//#include <math.h>
//#include "s_6dof.h"
import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"gonum.org/v1/gonum/mat"
	"k8s.io/klog/v2"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/smilelinkd/digitalbow-mapper/pkg/common"
)

const (
	IP_A_x float64 = 0
	IP_A_y float64 = 0
	IP_A_z float64 = 50
)

// BowRTUConfig is the configurations of modbus RTU.
type BowRTUConfig struct {
	SerialName   string
	BaudRate     int
	DataBits     int
	StopBits     int
	Parity       string
	RS485Enabled bool
	Timeout      time.Duration
}

type TrackData struct {
	Size       int             `json:"size"`
	Frequency  int             `json:"frequency"`
	MatrixList [][4][4]float64 `json:"Matrix_list"`
	MatrixInit [4][4]float64   `json:"Matrix_init"`
	IPList     [][3]float64    `json:"IP_list"`
	LCList     [][3]float64    `json:"LC_list"`
	RCList     [][3]float64    `json:"RC_list"`
}

type Client interface {
	Init()
	Close() (err error)
	GetStatus() interface{}
	Execute(movements []float64, clylen []float64)
}

type BowClient struct {
	Config BowRTUConfig
}

func (bowClient BowClient) Init() {
	C.SixDOFInit()
	klog.V(2).Info("init success...")
}

func (bowClient BowClient) Close() (err error) {
	//a := float32(1.1)
	//b := float32(1.1)
	//C.add((*C.float)(&a), (*C.float)(&b))
	println("close...")
	return nil
}

func (bowClient BowClient) GetStatus() interface{} {
	//a := float32(2.2)
	//b := float32(2.2)
	////C.add((*C.float)(&a), (*C.float)(&b))
	println("get status...")
	return nil
}

func (bowClient BowClient) Execute(movements []float32, clylen []float32) {
	klog.V(2).Infof("execute input %v", movements)
	C.SoluteCylinderLength((*C.float)(&movements[0]), (*C.float)(&clylen[0]))
}

// DigitalbowClient is the structure for modbus client.
type DigitalbowClient struct {
	Client       BowClient
	Status       common.DeviceStatus
	Movements    map[string]TrackData
	mu           sync.Mutex
	Transform_AU *mat.Dense
	Rotation_AU  *mat.Dense
}

/*
* In bow RTU mode, devices could connect to one serial port on RS485. However,
* the serial port doesn't support paralleled visit, and for one tcp device, it also doesn't support
* paralleled visit, so we expect one client for one port.
 */
var clients map[string]*DigitalbowClient

func newRTUClient(config BowRTUConfig) *DigitalbowClient {
	if clients == nil {
		clients = make(map[string]*DigitalbowClient)
	}

	if client, ok := clients[config.SerialName]; ok {
		return client
	}

	//# 平移向量:
	UO_A_x := IP_A_x
	UO_A_y := IP_A_y
	UO_A_z := IP_A_z

	//# U->A的旋转矩阵，变换矩阵:
	//# 3X3 旋转矩阵
	Rotation_AU := mat.NewDense(3, 3, nil)
	Rotation_AU.SetRow(0, []float64{
		1, 0, 0,
	})
	Rotation_AU.SetRow(1, []float64{
		0, math.Cos(math.Pi * 90 / 180), -math.Sin(math.Pi * 90 / 180),
	})
	Rotation_AU.SetRow(2, []float64{
		0, math.Sin(math.Pi * 90 / 180), math.Cos(math.Pi * 90 / 180),
	})

	Transform_AU := mat.NewDense(4, 4, nil)
	Transform_AU.SetRow(0, []float64{
		1, 0, 0, UO_A_x,
	})
	Transform_AU.SetRow(1, []float64{
		0, math.Cos(math.Pi * 90 / 180), -math.Sin(math.Pi * 90 / 180), UO_A_y,
	})
	Transform_AU.SetRow(2, []float64{
		0, math.Sin(math.Pi * 90 / 180), math.Cos(math.Pi * 90 / 180), UO_A_z,
	})
	Transform_AU.SetRow(3, []float64{
		0, 0, 0, 1,
	})

	client := DigitalbowClient{
		Status: common.StatusReady,
		Client: BowClient{
			Config: config,
		},
		Movements:    make(map[string]TrackData, 0),
		Transform_AU: Transform_AU,
		Rotation_AU:  Rotation_AU,
	}

	clients[config.SerialName] = &client
	return &client
}

// NewClient allocate and return a modbus client.
// Client type includes TCP and RTU.
func NewClient(config interface{}) (*DigitalbowClient, error) {
	switch c := config.(type) {
	case BowRTUConfig:
		return newRTUClient(c), nil
	default:
		return &DigitalbowClient{}, errors.New("Wrong type")
	}
}

// GetStatus get device status.
// Now we could only get the connection status.
func (c *DigitalbowClient) GetStatus() common.DeviceStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Status
}

// SetStatus set device status.
// Now we could only get the connection status.
func (c *DigitalbowClient) SetStatus(status common.DeviceStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Status = status
}

func (c *DigitalbowClient) DownloadResult(path, segment string) (err error) {
	var movement TrackData
	obsConfig := GetObs()
	obsClient, err := obs.New(obsConfig["AK"], obsConfig["SK"], obsConfig["URI"])
	if err != nil {
		return err
	}
	input := &obs.GetObjectInput{}
	input.Bucket = obsConfig["NAME"]
	input.Key = fmt.Sprintf("%s/%s_track.json", path, segment)
	content := make([]byte, 0)
	output, err := obsClient.GetObject(input)
	if err == nil {
		// output.Body 在使用完毕后必须关闭，否则会造成连接泄漏。
		defer output.Body.Close()
		fmt.Printf("Get object(%s) under the bucket(%s) successful!\n", input.Key, input.Bucket)
		fmt.Printf("StorageClass:%s, ETag:%s, ContentType:%s, ContentLength:%d, LastModified:%s\n",
			output.StorageClass, output.ETag, output.ContentType, output.ContentLength, output.LastModified)
		// 读取对象内容
		p := make([]byte, 1024)
		var readErr error
		var readCount int
		for {
			readCount, readErr = output.Body.Read(p)
			if readCount > 0 {
				content = append(content, p[:readCount]...)
			}
			if readErr != nil {
				break
			}
		}
	}

	if obsError, ok := err.(obs.ObsError); ok {
		fmt.Println("An ObsError was found, which means your request sent to OBS was rejected with an error response.")
		fmt.Println(obsError.Error())
		return
	}
	err = json.Unmarshal(content, &movement)
	if err != nil {
		fmt.Println("unmarshall error", err.Error())
		return
	}
	defer obsClient.Close()
	c.Movements[segment] = movement
	return nil
}

// Set set register.
func (c *DigitalbowClient) Set(registerType string, addr uint16, value uint16) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	klog.V(1).Info("Set:", registerType, addr, value)

	klog.V(1).Info("Set result:", err, results)
	return results, err
}

type EulerAngle struct {
	roll, pitch, yaw float64
}

func matrixToEuler(m [][]float64) EulerAngle {
	var euler EulerAngle

	if m[2][0] == -1 {
		euler.pitch = math.Pi / 2
		euler.roll = math.Atan2(m[0][1], m[0][2])
		euler.yaw = 0
	} else if m[2][0] == 1 {
		euler.pitch = -math.Pi / 2
		euler.roll = math.Atan2(-m[0][1], -m[0][2])
		euler.yaw = 0
	} else {
		// Roll (X-axis rotation)
		euler.roll = math.Atan2(m[2][1], m[2][2])
		// Pitch (Y-axis rotation)
		euler.pitch = math.Asin(-m[2][0])
		// Yaw (Z-axis rotation)
		euler.yaw = math.Atan2(m[1][0], m[0][0])
	}

	// Convert to degrees
	euler.roll = euler.roll * 180.0 / math.Pi
	euler.pitch = euler.pitch * 180.0 / math.Pi
	euler.yaw = euler.yaw * 180.0 / math.Pi

	return euler
}

func (c *DigitalbowClient) GetBowDataformat(trackData [4][4]float64, matrixInit [4][4]float64) []float64 {
	result := make([]float64, 6)

	trackMatrix := mat.NewDense(4, 4, nil)
	for i := 0; i < 4; i++ {
		trackMatrix.SetRow(i, trackData[i][:])
	}
	trackInitMatrix := mat.NewDense(4, 4, nil)
	for i := 0; i < 4; i++ {
		trackInitMatrix.SetRow(i, matrixInit[i][:])
	}
	var loopMatrix mat.Dense
	loopMatrix.Add(trackMatrix, trackInitMatrix)
	//#--------- 分别计算A坐标系下的欧拉角，和 平移向量
	//# 旋转矩阵 3X3
	rotateMatrix := [][]float64{
		loopMatrix.RawRowView(0)[0:3],
		loopMatrix.RawRowView(1)[0:3],
		loopMatrix.RawRowView(2)[0:3],
	}
	//# 旋转矩阵 转换 欧拉角
	eulerAngle := matrixToEuler(rotateMatrix)
	//# A坐标系下：计算欧拉角，这一步不能提前，必须在欧拉角算出来后，把U坐标系下的欧拉角换到A坐标系下
	var eulerA mat.Dense
	eulerAngleMatrix := mat.NewDense(3, 1, []float64{
		eulerAngle.roll, eulerAngle.pitch, eulerAngle.yaw,
	})
	eulerA.Mul(c.Rotation_AU, eulerAngleMatrix)
	//# A坐标系下：计算平移向量，位移单位要求为m。 之前是mm，所以要除1000
	var vectorA mat.Dense
	vectorA.Mul(c.Transform_AU, &loopMatrix)

	result[0] = eulerA.At(0, 0)
	result[1] = eulerA.At(1, 0)
	result[2] = eulerA.At(2, 0)
	result[3] = vectorA.At(0, 3) / 1000
	result[4] = vectorA.At(1, 3) / 1000
	result[4] = vectorA.At(2, 3) / 1000
	return result
}

func (c *DigitalbowClient) AssembleSerialData(moves []float32) []byte {
	b := []byte{0x55, 0xAA, 0x13, 0xFF, 0xF3}
	sum := 0x13 + 0xFF + 0xF3
	for i, item := range moves {
		id := byte(i + 1)
		number1 := int32((item - 0.1569) * 40000)
		number1_1 := byte(number1)
		number1_2 := byte(number1 >> 8)
		b = append(b, id, number1_1, number1_2)
		sum += int(id) + int(number1_2) + int(number1_1)
	}
	b = append(b, byte(sum))
	return b
}

func (c *DigitalbowClient) RandomGetCylen(i int) []float32 {
	if i%3 == 0 {
		return []float32{2, 0, 0, 0, 0, 0}
	} else if i%3 == 1 {
		return []float32{-5, 5, -5, 0.01, -0.01, 0.01}
	} else {
		return []float32{5, -5, 5, -0.01, 0.01, -0.01}
	}
}

func (c *DigitalbowClient) ResetToZero() []byte {
	clylen := make([]float32, 6)
	bowResult := []float32{0, 0, 0, 0, 0, 0}
	c.Client.Execute(bowResult, clylen)
	klog.V(2).Infof("execute output %v", clylen)
	writeMessage := c.AssembleSerialData(clylen)
	return writeMessage
}
