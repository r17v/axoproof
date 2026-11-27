package scanners

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveGitDiffFiles returns relative slash-normalized paths changed in the working tree (or HEAD~1 if clean).
func ResolveGitDiffFiles(workDir string) map[string]bool {
	files := map[string]bool{}

	if out, err := exec.Command("git", "-C", workDir, "status", "--porcelain").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if len(line) < 4 {
				continue
			}
			rawPath := strings.TrimSpace(line[3:])
			if idx := strings.Index(rawPath, " -> "); idx >= 0 {
				rawPath = strings.TrimSpace(rawPath[idx+4:])
			}
			rawPath = strings.Trim(rawPath, `"`)
			if rawPath != "" {
				files[filepath.ToSlash(rawPath)] = true
			}
		}
	}

	if len(files) == 0 {
		if out, err := exec.Command("git", "-C", workDir, "diff", "--name-only", "HEAD~1", "HEAD").Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				p := strings.TrimSpace(line)
				if p != "" {
					files[filepath.ToSlash(p)] = true
				}
			}
		}
	}

	return files
}
