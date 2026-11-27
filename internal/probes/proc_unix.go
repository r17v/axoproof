//go:build !windows

package probes

import (
	"os/exec"
	"syscall"
)

func buildShellCommand(workDir string, commandLine string) (*exec.Cmd, func()) {
	cmd := exec.Command("sh", "-c", commandLine)
	cmd.Dir = workDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stop := func() {
		if cmd.Process == nil {
			return
		}
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return cmd, stop
}
