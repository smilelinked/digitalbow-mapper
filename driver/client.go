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

	"k8s.io/klog/v2"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/smilelinkd/digitalbow-mapper/pkg/common"
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
	Client    BowClient
	Status    common.DeviceStatus
	Movements map[string]TrackData
	mu        sync.Mutex
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

	client := DigitalbowClient{
		Status: common.StatusReady,
		Client: BowClient{
			Config: config,
		},
		Movements: make(map[string]TrackData, 0),
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

func (c *DigitalbowClient) GetBowDataformat(trackData [4][4]float64) []float32 {
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

func (c *DigitalbowClient) AssembleSerialData(moves []float32) []byte {
	b := []byte{0x55, 0xAA, 0x13, 0xFF, 0xF3}
	sum := 0x13 + 0xFF + 0xF3
	for i, item := range moves {
		id := byte(i + 1)
		number1 := int32((item - 0.1569) * 40000)
		number1_2 := byte(number1)
		number1_1 := byte(number1 >> 8)
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
