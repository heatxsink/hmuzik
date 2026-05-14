//go:build !linux

package recentlyadded

import (
	"os"
	"time"
)

func fileCtime(info os.FileInfo) time.Time { return info.ModTime() }
