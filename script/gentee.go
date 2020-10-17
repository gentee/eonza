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
	Total   int64
	Current int64
	Source  string
	Dest    string
	Percent int64
	Remain  time.Duration

	start   time.Time
	updated time.Time
	handle  ProgressFunc
}

type ProgressInfo struct {
	ID      uint32 `json:"id"`
	Type    int    `json:"type"`
	Total   string `json:"total"`
	Current string `json:"current"`
	Source  string `json:"source"`
	Dest    string `json:"dest"`
	Percent int64  `json:"percent"`
	Remain  string `json:"remain"`
}

type ProgressFunc func(Progress) bool

type ProgressReader struct {
	Progress
	reader io.Reader
}

func ProgressHandle(prog Progress) bool {
	ChProgress <- prog
	return true
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

func NewProgress(r io.Reader, total int64, ptype int, handle ProgressFunc) *ProgressReader {
	now := time.Now()
	prog := Progress{
		ID:      lib.RndNum(),
		Total:   total,
		Type:    ptype,
		start:   now,
		updated: now,
		handle:  handle,
	}
	return &ProgressReader{
		Progress: prog,
		reader:   r,
	}
}

var ChProgress chan Progress

func (progress *ProgressReader) Read(data []byte) (n int, err error) {
	n, err = progress.reader.Read(data)
	if err == nil && n > 0 {
		var percent int64
		progress.Current += int64(n)
		ratio := float64(progress.Current) / float64(progress.Total)
		if progress.Current >= progress.Total {
			percent = 100
		} else {
			percent = int64(100.0 * ratio)
		}
		if /*percent != progress.Percent &&*/ time.Since(progress.updated) > 500*time.Millisecond {
			/*				dif := progress.Current - progress.prevTotal
							speed := float64(dif) / float64(since)
							progress.Remain = time.Duration(float64(progress.Total-progress.Current) / speed).Round(time.Second)*/
			remain := time.Duration(float64(time.Since(progress.start)) * (1 - ratio) / ratio).Round(time.Second)
			if percent != progress.Percent || remain != progress.Remain {
				progress.Percent = percent
				progress.Remain = remain
				progress.updated = time.Now()
				progress.handle(progress.Progress)
			}
		}
	}
	return
}

func (progress *Progress) String() (string, error) {
	var remain string
	if progress.Remain <= 24*time.Hour {
		remain = progress.Remain.String()
	}

	data, err := json.Marshal(ProgressInfo{
		ID:      progress.ID,
		Type:    progress.Type,
		Total:   SizeToStr(progress.Total),
		Current: SizeToStr(progress.Current),
		Percent: progress.Percent,
		Remain:  remain,
		Source:  progress.Source,
		Dest:    progress.Dest,
	})
	return string(data), err
}

func (progress *Progress) Complete() {
	if progress.start == progress.updated {
		return
	}
	progress.Percent = 100
	progress.handle(*progress)
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

	prog := NewProgress(srcFile, finfo.Size(), ProgCopy, ProgressHandle)
	prog.Source = src
	prog.Dest = dest
	ret, err := io.Copy(destFile, prog)
	prog.Complete()
	destFile.Chmod(finfo.Mode())
	return ret, err
}
