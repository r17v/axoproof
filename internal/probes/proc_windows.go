//go:build windows

package probes

import (
	"os/exec"
	"strconv"
)

func buildShellCommand(workDir string, commandLine string) (*exec.Cmd, func()) {
	cmd := exec.Command("cmd.exe", "/C", commandLine)
	cmd.Dir = workDir
	stop := func() {
		if cmd.Process == nil {
			return
		}
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	return cmd, stop
}
