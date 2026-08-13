package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const lineHistoryPrefix = "COMMIT\x00"

// GetLineCountHistory returns the tracked text line count after each commit on
// the first-parent history of HEAD. Binary files are excluded by git numstat.
func GetLineCountHistory(ctx context.Context, gitDir string) ([]LineCountPoint, error) {
	cmd := exec.CommandContext(
		ctx,
		"git",
		"--git-dir="+gitDir,
		"log",
		"--first-parent",
		"--reverse",
		"--root",
		"--diff-merges=first-parent",
		"--format=COMMIT%x00%H%x00%ct",
		"--numstat",
		"HEAD",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git line history failed: %w", err)
	}
	return parseLineCountHistory(out)
}

func parseLineCountHistory(data []byte) ([]LineCountPoint, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// A numstat line can contain a long renamed path.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	var result []LineCountPoint
	total := 0
	var current *LineCountPoint
	appendCurrent := func() {
		if current == nil {
			return
		}
		current.Lines = total
		result = append(result, *current)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, lineHistoryPrefix) {
			appendCurrent()
			fields := strings.Split(line, "\x00")
			if len(fields) != 3 {
				return nil, fmt.Errorf("invalid git line history header")
			}
			unixSeconds, err := strconv.ParseInt(fields[2], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid git line history timestamp: %w", err)
			}
			current = &LineCountPoint{SHA: fields[1], Date: time.Unix(unixSeconds, 0).UTC()}
			continue
		}

		if current == nil || line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 2 || fields[0] == "-" || fields[1] == "-" {
			continue
		}
		additions, addErr := strconv.Atoi(fields[0])
		deletions, deleteErr := strconv.Atoi(fields[1])
		if addErr != nil || deleteErr != nil {
			continue
		}
		total += additions - deletions
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan git line history: %w", err)
	}
	appendCurrent()
	return result, nil
}
