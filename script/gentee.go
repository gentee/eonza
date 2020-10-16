// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"encoding/json"
	"eonza/lib"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gentee/gentee"
	"github.com/gentee/gentee/core"
	"github.com/gentee/gentee/vm"
	"gopkg.in/yaml.v2"
)

func YamlToMap(in string) (*core.Map, error) {
	var (
		tmp map[string]string
		ret interface{}
		err error
	)
	if err = yaml.Unmarshal([]byte(in), &tmp); err != nil {
		return nil, err
	}
	ret, err = gentee.Go2GenteeType(tmp, `map.str`)
	if err != nil {
		return nil, err
	}
	return ret.(*core.Map), nil
}

// UnsetEnv unsets the environment variable
func UnsetEnv(rt *vm.Runtime, name string) error {
	if rt.Owner.Settings.IsPlayground {
		// restore in gentee ErrorText(ErrPlayEnv))
		return fmt.Errorf(`[Playground] setting the environment variable is disabled`)
	}
	return os.Unsetenv(name)
}

// remove after gentee update
// replaced GetEnvironment to GetEnv in Set Variable command
func GetEnv(name string) string {
	return os.Getenv(name)
}

const (
	ProgCopy = iota
)

type Progress struct {
	ID      uint32
	Type    int
	Summary int64
	CurSize int64
	Source  string
	Dest    string
	Percent int64
	Remain  time.Duration
}

type ProgressInfo struct {
	ID      uint32 `json:"id"`
	Type    int    `json:"type"`
	Summary string `json:"summary"`
	CurSize string `json:"cursize"`
	Source  string `json:"source,omitempty"`
	Dest    string `json:"dest,omitempty"`
	Percent int64  `json:"percent"`
	Remain  string `json:"remain"`
}

type ProgressReader struct {
	Progress
	reader  io.Reader
	start   time.Time
	updated time.Time
	ch      chan Progress
}

const (
	sizeB  int64 = 1
	sizeKB int64 = 1 << (10 * iota)
	sizeMB
	sizeGB
	sizeTB
)

func SizeToStr(size int64) string {
	var (
		base int64
		ext  string
	)
	switch {
	case size >= sizeTB:
		base = sizeTB
		ext = "TB"
	case size >= sizeGB:
		base = sizeGB
		ext = "GB"
	case size >= sizeMB:
		base = sizeMB
		ext = "MB"
	case size >= sizeKB:
		base = sizeKB
		ext = "KB"
	default:
		base = sizeB
		ext = "B"
	}
	return fmt.Sprintf("%.2f", float64(size)/float64(base)) + ext
}

func NewProgress(r io.Reader, size int64, ptype int) *ProgressReader {
	now := time.Now()
	return &ProgressReader{
		Progress: Progress{ID: lib.RndNum(), Summary: size, Type: ptype},
		reader:   r,
		start:    now,
		updated:  now,
	}
}

var ChProgress chan Progress

func (progress *ProgressReader) Read(data []byte) (n int, err error) {
	n, err = progress.reader.Read(data)
	if err == nil {
		var percent int64
		progress.CurSize += int64(n)
		if progress.CurSize > 0 {
			ratio := float64(progress.CurSize) / float64(progress.Summary)
			if progress.CurSize >= progress.Summary {
				percent = 100
			} else {
				percent = int64(100.0 * ratio)

			}
			if /*percent != progress.Percent &&*/ time.Since(progress.updated) > 500*time.Millisecond {
				progress.Remain = time.Duration(float64(time.Since(progress.start)) * (1 - ratio) / ratio).Round(time.Second)
				progress.Percent = percent
				progress.Update()
			}
		}
	}
	return
}

func (progress *Progress) String() (string, error) {
	data, err := json.Marshal(ProgressInfo{
		ID:      progress.ID,
		Type:    progress.Type,
		Summary: SizeToStr(progress.Summary),
		CurSize: SizeToStr(progress.CurSize),
		Percent: progress.Percent,
		Remain:  progress.Remain.String(),
		Source:  progress.Source,
		Dest:    progress.Dest,
	})
	return string(data), err
}

func (progress *ProgressReader) Update() {
	progress.updated = time.Now()
	progress.ch <- progress.Progress
}

func (progress *ProgressReader) Complete() {
	if progress.start == progress.updated {
		return
	}
	progress.Percent = 100
	progress.Update()
	return
}

// For testing
func CopyFileEx(rt *vm.Runtime, src, dest string) (int64, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	finfo, err := srcFile.Stat()
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer destFile.Close()

	prog := NewProgress(srcFile, finfo.Size(), ProgCopy)
	prog.Source = src
	prog.Dest = dest
	prog.ch = ChProgress
	ret, err := io.Copy(destFile, prog)
	prog.Complete()
	destFile.Chmod(finfo.Mode())
	return ret, err
}
