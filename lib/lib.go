// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package lib

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/kataras/golog"
)

// HTTPConfig stores web-server settings
type HTTPConfig struct {
	Port  int    `yaml:"port"`  // if empty, then DefPort
	Open  bool   `yaml:"open"`  // if true then host is opened
	Theme string `yaml:"theme"` // theme of web interface. if it is empty - DefTheme
}

// AppPath returns the full path of the current application file
func AppPath(path ...string) (ret string) {
	var err error
	if len(path) == 0 {
		//		ret = os.Args[0]
		if ret, err = os.Executable(); err != nil {
			golog.Fatal(err)
		}
	} else {
		ret = path[0]
	}
	if !filepath.IsAbs(ret) {
		if ret, err = filepath.Abs(ret); err != nil {
			golog.Fatal(err)
		}
	}
	return
}

// ChangeExt changes the extension of the file
func ChangeExt(path string, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + `.` + ext
}

// Open opens the corresponding app for filename
func Open(filename string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", filename).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", filename).Start()
	case "darwin":
		err = exec.Command("open", filename).Start()
	default:
		err = fmt.Errorf(`unsupported platform`)
	}
	return err
}

// UniqueName returns a random string
func UniqueName(count int) string {
	b := make([]rune, count)
	var alphabet = []rune("0123456789abcdefghijklmnopqrstuvwxyz")
	alen := len(alphabet)
	for i := range b {
		b[i] = alphabet[rand.Intn(alen)]
	}
	if b[0] < 'a' {
		b[0] = alphabet[10+rand.Intn(alen-10)]
	}
	return string(b)
}

// ValidateSysName checks the system name
func ValidateSysName(value string) bool {
	re, _ := regexp.Compile(`^[a-z][a-z\d\._-]*$`)
	return re.MatchString(value)
}

func RndNum() uint32 {
	return rand.Uint32()
}

func init() {
	rand.Seed(time.Now().Unix())
}

func ClearCarriage(input string) string {
	var start int
	runes := []rune(string(strings.TrimRight(input, "\r")))
	out := make([]rune, 0, len(runes))
	for _, char := range []rune(runes) {
		if char == 0xd {
			out = out[:start]
		} else {
			out = append(out, char)
			if char == 0xa {
				start = len(out)
			}
		}
	}
	return string(out)
}
