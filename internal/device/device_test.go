package device

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// fake answers commands from a table keyed by "name arg...", recording each
// call. Missing keys succeed with empty output.
type fake struct {
	out   map[string]string
	fail  map[string]error
	calls []string
}

func (f *fake) run(_ context.Context, name string, args ...string) (string, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	if err := f.fail[key]; err != nil {
		return "", err
	}
	return f.out[key], nil
}

const (
	findmnt   = "findmnt --noheadings --output SOURCE --mountpoint /run/media/u/IPOD"
	lsblkPart = "lsblk --noheadings --nodeps --output PKNAME /dev/sda2"
	unmount   = "udisksctl unmount --block-device /dev/sda2"
	powerOff  = "udisksctl power-off --block-device /dev/sda"
)

func TestEject_UnmountsPartitionThenPowersOffDisk(t *testing.T) {
	f := &fake{out: map[string]string{findmnt: "/dev/sda2", lsblkPart: "sda"}}
	if err := eject(context.Background(), "/run/media/u/IPOD", f.run); err != nil {
		t.Fatal(err)
	}
	want := []string{findmnt, lsblkPart, unmount, powerOff}
	if !reflect.DeepEqual(f.calls, want) {
		t.Errorf("calls:\nwant %q\ngot  %q", want, f.calls)
	}
}

func TestEject_WholeDiskFilesystem(t *testing.T) {
	f := &fake{out: map[string]string{
		"findmnt --noheadings --output SOURCE --mountpoint /m": "/dev/sdb",
	}}
	if err := eject(context.Background(), "/m", f.run); err != nil {
		t.Fatal(err)
	}
	if last := f.calls[len(f.calls)-1]; last != "udisksctl power-off --block-device /dev/sdb" {
		t.Errorf("powered off %q", last)
	}
}

func TestEject_NotAMountPoint(t *testing.T) {
	f := &fake{fail: map[string]error{findmnt: errors.New("exit status 1")}}
	err := eject(context.Background(), "/run/media/u/IPOD", f.run)
	if err == nil || !strings.Contains(err.Error(), "not a mount point") {
		t.Fatalf("want not-a-mount-point error, got %v", err)
	}
	if len(f.calls) != 1 {
		t.Errorf("ran more than findmnt: %q", f.calls)
	}
}

func TestEject_UnmountFailureSkipsPowerOff(t *testing.T) {
	f := &fake{
		out:  map[string]string{findmnt: "/dev/sda2", lsblkPart: "sda"},
		fail: map[string]error{unmount: errors.New("target is busy")},
	}
	err := eject(context.Background(), "/run/media/u/IPOD", f.run)
	if err == nil || !strings.Contains(err.Error(), "do not unplug") {
		t.Fatalf("want do-not-unplug error, got %v", err)
	}
	for _, c := range f.calls {
		if c == powerOff {
			t.Error("powered off after failed unmount")
		}
	}
}
