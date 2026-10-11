// Package device ejects removable media through udisks, the same way the
// desktop does: unmount the filesystem, then power off the drive so it is safe
// to unplug.
package device

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// runner runs an external command and returns its trimmed stdout.
type runner func(ctx context.Context, name string, args ...string) (string, error)

func execRun(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Eject unmounts the filesystem mounted exactly at mountPoint and powers off
// the drive it lives on. It needs findmnt, lsblk and udisksctl (Linux).
func Eject(ctx context.Context, mountPoint string) error {
	return eject(ctx, mountPoint, execRun)
}

func eject(ctx context.Context, mountPoint string, run runner) error {
	part, err := run(ctx, "findmnt", "--noheadings", "--output", "SOURCE", "--mountpoint", mountPoint)
	if err != nil || part == "" {
		return fmt.Errorf("%s is not a mount point; nothing to eject: %w", mountPoint, errOrEmpty(err))
	}
	// A whole-disk filesystem has no parent; power off the device itself.
	disk := part
	parent, err := run(ctx, "lsblk", "--noheadings", "--nodeps", "--output", "PKNAME", part)
	if err != nil {
		return fmt.Errorf("finding the drive for %s: %w", part, err)
	}
	if parent != "" {
		disk = "/dev/" + parent
	}
	if _, err := run(ctx, "udisksctl", "unmount", "--block-device", part); err != nil {
		return fmt.Errorf("unmount %s failed, do not unplug: %w", part, err)
	}
	if _, err := run(ctx, "udisksctl", "power-off", "--block-device", disk); err != nil {
		return fmt.Errorf("%s unmounted but power-off of %s failed: %w", part, disk, err)
	}
	return nil
}

var errEmpty = errors.New("findmnt returned no source")

func errOrEmpty(err error) error {
	if err != nil {
		return err
	}
	return errEmpty
}
