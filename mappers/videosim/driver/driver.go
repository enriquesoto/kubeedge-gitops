package driver

import (
	"fmt"
	"io"
	"os/exec"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

const (
	// analysis resolution: small on purpose — motion detection by frame
	// differencing doesn't need full resolution and this keeps CPU low
	frameW = 160
	frameH = 120

	// per-pixel absolute gray delta considered "changed"
	pixelDelta = 25
)

func NewClient(protocol ProtocolConfig) (*CustomizedClient, error) {
	if protocol.VideoPath == "" {
		return nil, fmt.Errorf("videosim: configData.videoPath is required")
	}
	if protocol.SampleFPS <= 0 {
		protocol.SampleFPS = 4
	}
	if protocol.MotionThreshold <= 0 {
		protocol.MotionThreshold = 0.02
	}
	client := &CustomizedClient{
		ProtocolConfig: protocol,
		deviceMutex:    sync.Mutex{},
		stopCh:         make(chan struct{}),
	}
	return client, nil
}

func (c *CustomizedClient) InitDevice() error {
	// Decode the looped mp4 to raw grayscale frames on stdout. -re paces
	// decoding at native speed so this behaves like a live feed.
	cmd := exec.Command("ffmpeg",
		"-stream_loop", "-1",
		"-re",
		"-i", c.VideoPath,
		"-vf", fmt.Sprintf("fps=%d,scale=%dx%d", c.SampleFPS, frameW, frameH),
		"-f", "rawvideo",
		"-pix_fmt", "gray",
		"-loglevel", "error",
		"-",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("videosim: stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("videosim: starting ffmpeg: %v", err)
	}
	c.ffmpegCmd = cmd
	c.online = true
	klog.Infof("videosim: ffmpeg started on %s (fps=%d threshold=%.3f)",
		c.VideoPath, c.SampleFPS, c.MotionThreshold)

	go c.analyzeLoop(stdout)
	return nil
}

// analyzeLoop reads raw gray frames and flags motion when the fraction of
// pixels that changed against the previous frame exceeds MotionThreshold.
func (c *CustomizedClient) analyzeLoop(r io.Reader) {
	frameSize := frameW * frameH
	prev := make([]byte, frameSize)
	curr := make([]byte, frameSize)
	havePrev := false

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}
		if _, err := io.ReadFull(r, curr); err != nil {
			klog.Warningf("videosim: frame read ended: %v", err)
			c.deviceMutex.Lock()
			c.online = false
			c.deviceMutex.Unlock()
			return
		}

		changed := 0
		if havePrev {
			for i := 0; i < frameSize; i++ {
				d := int(curr[i]) - int(prev[i])
				if d < 0 {
					d = -d
				}
				if d > pixelDelta {
					changed++
				}
			}
		}
		ratio := float64(changed) / float64(frameSize)

		c.deviceMutex.Lock()
		c.frameCount++
		c.motionDetected = havePrev && ratio > c.MotionThreshold
		if c.motionDetected {
			// map changed-pixel ratio to a 0.50-0.99 confidence band
			conf := 0.5 + ratio*10
			if conf > 0.99 {
				conf = 0.99
			}
			c.confidence = conf
		} else {
			c.confidence = 0
		}
		c.deviceMutex.Unlock()

		prev, curr = curr, prev
		havePrev = true
	}
}

func (c *CustomizedClient) GetDeviceData(visitor *VisitorConfig) (interface{}, error) {
	c.deviceMutex.Lock()
	defer c.deviceMutex.Unlock()
	switch visitor.PropertyName {
	case "status":
		if c.online {
			return "online", nil
		}
		return "offline", nil
	case "motionDetected":
		return c.motionDetected, nil
	case "frameCount":
		return c.frameCount, nil
	case "confidence":
		return fmt.Sprintf("%.2f", c.confidence), nil
	default:
		return nil, fmt.Errorf("videosim: unknown property %q", visitor.PropertyName)
	}
}

func (c *CustomizedClient) DeviceDataWrite(visitor *VisitorConfig, deviceMethodName string, propertyName string, data interface{}) error {
	return fmt.Errorf("videosim: all camera properties are read-only")
}

func (c *CustomizedClient) SetDeviceData(data interface{}, visitor *VisitorConfig) error {
	return fmt.Errorf("videosim: all camera properties are read-only")
}

func (c *CustomizedClient) StopDevice() error {
	close(c.stopCh)
	if c.ffmpegCmd != nil && c.ffmpegCmd.Process != nil {
		_ = c.ffmpegCmd.Process.Kill()
		_ = c.ffmpegCmd.Wait()
	}
	c.deviceMutex.Lock()
	c.online = false
	c.deviceMutex.Unlock()
	return nil
}

func (c *CustomizedClient) GetDeviceStates() (string, error) {
	c.deviceMutex.Lock()
	defer c.deviceMutex.Unlock()
	if !c.online {
		return common.DeviceStatusDisCONN, nil
	}
	return common.DeviceStatusOK, nil
}

func (c *CustomizedClient) AnomalyDetectionProcess(req *AnomalyDetectionRequest) error {
	return nil
}
