//go:build linux

package recentlyadded

import (
	"os"
	"syscall"
	"time"
)

func fileCtime(info os.FileInfo) time.Time {
	if s, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(s.Ctim.Sec, s.Ctim.Nsec)
	}
	return info.ModTime()
}
