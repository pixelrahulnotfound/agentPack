// Package sandbox shells out to the plain Docker CLI — no custom runtime.
package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Available reports whether the docker CLI is on PATH and the daemon responds.
func Available() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker CLI not found on PATH")
	}
	if out, err := exec.Command("docker", "info", "--format", "{{.ServerVersion}}").CombinedOutput(); err != nil {
		return fmt.Errorf("docker daemon unreachable: %s", string(out))
	}
	return nil
}

// RunArgs builds the `docker run` invocation for an agent directory.
// It mounts the agent dir at /work and the memory dir (if any) at /memory.
func RunArgs(agentDir, image string, extraArgs []string) []string {
	abs, _ := filepath.Abs(agentDir)
	args := []string{"run", "--rm", "-i", "-v", abs + ":/work", "-w", "/work"}
	args = append(args, extraArgs...)
	args = append(args, image)
	return args
}

// Run executes docker with args, wiring stdio through. Used by `agentpack run`.
func Run(agentDir, image string, extraArgs []string) error {
	if err := Available(); err != nil {
		return err
	}
	args := RunArgs(agentDir, image, extraArgs)
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run failed: %w", err)
	}
	return nil
}
