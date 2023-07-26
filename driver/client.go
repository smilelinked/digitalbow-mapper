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
	fmt.Println("init success...")
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

// Get get register.
func (c *DigitalbowClient) Get(registerType string, addr uint16, quantity uint16) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	klog.V(2).Info("Get result: ", results)
	return results, err
}

// Set set register.
func (c *DigitalbowClient) Set(registerType string, addr uint16, value uint16) (results []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	klog.V(1).Info("Set:", registerType, addr, value)

	klog.V(1).Info("Set result:", err, results)
	return results, err
}
