package driver

import (
	"encoding/json"
	"os/exec"
	"sync"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

// CustomizedDev is the customized device configuration and client information.
type CustomizedDev struct {
	Instance         common.DeviceInstance
	CustomizedClient *CustomizedClient
}

type CustomizedClient struct {
	deviceMutex sync.Mutex
	ProtocolConfig

	// ffmpeg decode pipeline
	ffmpegCmd *exec.Cmd
	stopCh    chan struct{}

	// latest analysis results, guarded by deviceMutex
	frameCount     int64
	motionDetected bool
	confidence     float64
	online         bool
}

type ProtocolConfig struct {
	ProtocolName string `json:"protocolName"`
	ConfigData   `json:"configData"`
}

type ConfigData struct {
	// VideoPath is the .mp4 file the mapper decodes in a loop,
	// standing in for a live RTSP camera feed.
	VideoPath string `json:"videoPath"`
	// SampleFPS is how many frames per second are analyzed (default 4).
	SampleFPS int `json:"sampleFPS,omitempty"`
	// MotionThreshold is the fraction of changed pixels between two
	// consecutive frames above which motion is reported (default 0.02).
	MotionThreshold float64 `json:"motionThreshold,omitempty"`
}

type VisitorConfig struct {
	ProtocolName      string `json:"protocolName"`
	VisitorConfigData `json:"configData"`
}

type VisitorConfigData struct {
	DataType string `json:"dataType"`
	// PropertyName tells the driver which camera property this visitor reads:
	// status | motionDetected | frameCount | confidence
	PropertyName string `json:"propertyName"`
}

type AnomalyDetectionRequest struct {
	Enabled                bool            `json:"enabled"`
	VisitorConfig          VisitorConfig   `json:"visitorConfig"`
	AnomalyDetectionConfig json.RawMessage `json:"anomalyDetectionConfig"`
	Data                   interface{}     `json:"data"`
}
