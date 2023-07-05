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

/*
static float add(float *a, float *b) {
    return *a + *b;
}
*/
import "C"
import (
	"errors"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/mappers-go/mappers/common"
)

// BowRTU is the configurations of modbus RTU.
type BowRTU struct {
	SlaveID      byte
	SerialName   string
	BaudRate     int
	DataBits     int
	StopBits     int
	Parity       string
	RS485Enabled bool
	Timeout      time.Duration
}

type Client interface {
	Init(a, b float32) (err error)
	Close() (err error)
	GetStatus() interface{}
	Execute(movements []float32, clylen []float32)
}

type BowClient struct {
}

func (bowClient BowClient) Init(a, b *float32) (err error) {
	C.add((*C.float)(a), (*C.float)(b))
	println("init...")
	return nil
}

func (bowClient BowClient) Close() (err error) {
	a := float32(1.1)
	b := float32(1.1)
	C.add((*C.float)(&a), (*C.float)(&b))
	println("close...")
	return nil
}

func (bowClient BowClient) GetStatus() interface{} {
	a := float32(2.2)
	b := float32(2.2)
	C.add((*C.float)(&a), (*C.float)(&b))
	println("get status...")
	return nil
}

func (bowClient BowClient) Execute(movements []float32, clylen []float32) {
	a := float32(3.3)
	b := float32(3.3)
	C.add((*C.float)(&a), (*C.float)(&b))
	println("execute...")
}

// DigitalbowClient is the structure for modbus client.
type DigitalbowClient struct {
	Client BowClient
	mu     sync.Mutex
}

/*
* In bow RTU mode, devices could connect to one serial port on RS485. However,
* the serial port doesn't support paralleled visit, and for one tcp device, it also doesn't support
* paralleled visit, so we expect one client for one port.
 */
var clients map[string]*DigitalbowClient

func newRTUClient(config BowRTU) *DigitalbowClient {
	if clients == nil {
		clients = make(map[string]*DigitalbowClient)
	}

	if client, ok := clients[config.SerialName]; ok {
		return client
	}

	client := DigitalbowClient{}
	clients[config.SerialName] = &client
	return &client
}

// NewClient allocate and return a modbus client.
// Client type includes TCP and RTU.
func NewClient(config interface{}) (*DigitalbowClient, error) {
	switch c := config.(type) {
	case BowRTU:
		return newRTUClient(c), nil
	default:
		return &DigitalbowClient{}, errors.New("Wrong modbus type")
	}
}

// GetStatus get device status.
// Now we could only get the connection status.
func (c *DigitalbowClient) GetStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.Client.GetStatus()
	if err == nil {
		return common.DEVSTOK
	}
	return common.DEVSTDISCONN
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

// parity convert into the format that modbus driver requires.
func parity(ori string) string {
	var p string
	switch ori {
	case "even":
		p = "E"
	case "odd":
		p = "O"
	default:
		p = "N"
	}
	return p
}
