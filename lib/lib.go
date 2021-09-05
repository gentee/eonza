// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package lib

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
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
	Host      string      `yaml:"host"`      // if empty, then localhost
	Port      int         `yaml:"port"`      // if empty, then DefPort
	LocalPort int         `yaml:"localport"` // if empty, then define automatically
	Open      bool        `yaml:"open"`      // if true then host is opened
	Theme     string      `yaml:"theme"`     // theme of web interface. if it is empty - DefTheme
	JWTKey    string      `yaml:"jwtkey"`    // Secret key for JWT token
	Cert      interface{} `yaml:"cert"`      // cert pem file
	Priv      interface{} `yaml:"priv"`      // private key pem file
}

// PlaygroundConfig stores the config of playgroundmode
type PlaygroundConfig struct {
	Dir     string `yaml:"dir"`     // path to the temporary folder if it's empty then TempDir is used.
	Summary int64  `yaml:"summary"` // all files size limit. By default, 10MB
	Files   int64  `yaml:"files"`   // count of files limit. By default, 100
	Size    int64  `yaml:"size"`    // file size limit. By default, 5MB
	Tasks   int64  `yaml:"tasks"`   // running task limit. By default, 1
}

var (
	privateIPBlocks []*net.IPNet
	ipPrivateList   = []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	}
	reSysName, _ = regexp.Compile(`^[a-z][a-z\d\._-]*$`)
)

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

func IdName(value string) string {
	out := strings.ReplaceAll(value, `.`, `_`)
	return strings.ReplaceAll(out, `-`, `_`)
}

// ValidateSysName checks the system name
func ValidateSysName(value string) bool {
	return reSysName.MatchString(value)
}

func RndNum() uint32 {
	return rand.Uint32()
}

func init() {
	rand.Seed(time.Now().Unix())

	for _, cidr := range ipPrivateList {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			golog.Error(err)
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
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

func ZipFiles(filename string, files []string) (err error) {
	var zipFile *os.File
	if zipFile, err = os.Create(filename); err != nil {
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		if err = AddFileToZip(zipWriter, file); err != nil {
			return
		}
	}
	return
}

func AddFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.Base(filename)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

func IsLocalhost(host, ipaddr string) bool {
	if host != `localhost` && host != `127.0.0.1` {
		return false
	}
	ip := net.ParseIP(ipaddr)
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func IsPrivate(host, ipaddr string) bool {
	var ip net.IP
	isPrivate := func() bool {
		for _, block := range privateIPBlocks {
			if block.Contains(ip) {
				return true
			}
		}
		return false
	}
	if host != `localhost` {
		ip = net.ParseIP(host)
		if ip == nil || !isPrivate() {
			return false
		}
	}
	ip = net.ParseIP(ipaddr)
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	return isPrivate()
}

func LocalGet(port int, url string) (body []byte, err error) {
	var (
		res *http.Response
	)
	res, err = http.Get(fmt.Sprintf(`http://localhost:%d/%s`, port, url))
	if err == nil {
		body, err = io.ReadAll(res.Body)
		res.Body.Close()
	}
	return
}

func LocalPost(port int, url string, data interface{}) (body []byte, err error) {
	var resp *http.Response

	jsonValue, err := json.Marshal(data)
	if err == nil {
		resp, err = http.Post(fmt.Sprintf("http://localhost:%d/%s", port, url),
			"application/json", bytes.NewBuffer(jsonValue))
		if err == nil {
			body, err = io.ReadAll(resp.Body)
			resp.Body.Close()
		}
	}
	return
}
