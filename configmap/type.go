package configmap

// ModbusVisitorConfig is the modbus register configuration.
type ModbusVisitorConfig struct {
	Register       string  `json:"register"`
	Offset         uint16  `json:"offset"`
	Limit          int     `json:"limit"`
	Scale          float64 `json:"scale,omitempty"`
	IsSwap         bool    `json:"isSwap,omitempty"`
	IsRegisterSwap bool    `json:"isRegisterSwap,omitempty"`
}

// BowProtocolConfig is the protocol configuration.
type BowProtocolConfig struct {
	SlaveID int16 `json:"slaveID,omitempty"`
}

// BowProtocolCommonConfig is the bow protocol configuration.
type BowProtocolCommonConfig struct {
	COM              COMStruct       `json:"com,omitempty"`
	CustomizedValues CustomizedValue `json:"customizedValues,omitempty"`
}

// CustomizedValue is the customized part for modbus protocol.
type CustomizedValue map[string]interface{}

// COMStruct is the serial configuration.
type COMStruct struct {
	SerialPort string `json:"serialPort"`
	BaudRate   int64  `json:"baudRate"`
	DataBits   int64  `json:"dataBits"`
	Parity     string `json:"parity"`
	StopBits   int64  `json:"stopBits"`
}

type DownloadRequest struct {
	Path    string `json:"path"`
	Segment string `json:"segment"`
}

type ExecuteRequest struct {
	Segment string    `json:"segment"`
	Random  bool      `json:"random"`
	Input   []float64 `json:"input"`
	Period  int       `json:"period"`
}
