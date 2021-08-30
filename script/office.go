// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package script

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const (
	DocxFile = iota
	OdtFile
)

var extTemplates = []string{
	"word/document.xml",
	"content.xml",
}

func parseDocx(in string, regexp *regexp.Regexp) (out string) {
	getleft := func(right int) int {
		left := strings.LastIndex(in[:right], "<w:r")
		if left > 0 {
			if in[left+4] != ' ' && in[left+4] != '>' {
				left := strings.LastIndex(in[:left-1], "<w:r")
				if left > 0 {
					return left
				}
			}
		}
		return 0
	}
	offset := regexp.FindAllStringSubmatchIndex(in, -1)
	shift := 0
	for i := range offset {
		left := getleft(offset[i][0])
		if left == 0 {
			continue
		}
		if sublen := offset[i][3] - offset[i][2]; sublen > 64 {
			continue
		}
		subleft := getleft(offset[i][2])
		subright := offset[i][3] + len(`</w:t></w:r>`)
		out += in[shift:left] + in[subleft:offset[i][2]] + `#` +
			in[offset[i][2]:offset[i][3]] + `#` + in[offset[i][3]:subright]
		shift = offset[i][1]
	}
	out += in[shift:]
	return
}

func ParseDoc(in string) string {
	var re *regexp.Regexp
	if strings.Contains(in, ">#</w:t>" /* "<w:t>{{</w:t>" */) {
		re = regexp.MustCompile(">#</w:t></w:r>.+?<w:t>(.+?)</w:t></w:r>.+?>#</w:t></w:r>")
		in = parseDocx(in, re)
	}
	/*	if strings.Contains(in, ">{</w:t>") {
		re = regexp.MustCompile(">{</w:t></w:r>.+?>{</w:t></w:r>.+?<w:t>(.+?)</w:t></w:r>.+?<w:t>}</w:t></w:r>.+?>}</w:t></w:r>")
		in = parseDocx(in, re)
	}*/
	return in
}

func replaceTemplate(src, dest string, template int) error {
	z, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer z.Close()

	target, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer target.Close()

	zw := zip.NewWriter(target)

	var size int64

	for _, f := range z.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		if size = f.FileInfo().Size(); size > 100000000 {
			return fmt.Errorf(`Too compressed big file`)
		}
		header, err := zip.FileInfoHeader(f.FileInfo())
		if err != nil {
			return err
		}
		var (
			isMacro bool
			out     string
		)
		if f.Name == extTemplates[template] {
			isMacro = true
			input := make([]byte, size)
			if read, err := io.ReadFull(rc, input); err != nil || read != int(size) {
				if err != nil {
					return err
				}
				return fmt.Errorf(`Decompressing error %d != %d`, read, size)
			}
			in := string(input)
			if template == DocxFile {
				in = ParseDoc(in)
			}
			out, err = Macro(in)
			if err != nil {
				return err
			}
			size = int64(len(out))
		}
		header.UncompressedSize64 = uint64(size)
		header.UncompressedSize = uint32(size)
		header.Name = f.Name
		item, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		if isMacro {
			if _, err = item.Write([]byte(out)); err != nil {
				return err
			}
		} else if _, err = io.Copy(item, rc); err != nil {
			return err
		}
	}
	zw.Close()
	return nil
}

func DocxTemplate(src string, dest string) error {
	return replaceTemplate(src, dest, DocxFile)
}

func OdtTemplate(src string, dest string) error {
	return replaceTemplate(src, dest, OdtFile)
}
