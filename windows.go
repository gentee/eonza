// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build windows

package main

import (
	"syscall"
	"unsafe"
)

// Useful link: https://github.com/gonutz/w32

type (
	DWORD  uint32
	HANDLE uintptr
	HWND   HANDLE
)

const (
	SW_HIDE = 0
)

var (
	kernel32, _                 = syscall.LoadLibrary("kernel32.dll")
	getConsoleWindow, _         = syscall.GetProcAddress(kernel32, "GetConsoleWindow")
	getCurrentProcessId, _      = syscall.GetProcAddress(kernel32, "GetCurrentProcessId")
	user32, _                   = syscall.LoadLibrary("user32.dll")
	getWindowThreadProcessId, _ = syscall.GetProcAddress(user32, "GetWindowThreadProcessId")
	showWindowAsync, _          = syscall.GetProcAddress(user32, "ShowWindowAsync")
)

func GetConsoleWindow() HWND {
	ret, _, _ := syscall.Syscall(uintptr(getConsoleWindow), 0, 0, 0, 0)
	return HWND(ret)
}

func GetWindowThreadProcessId(hwnd HWND) (HANDLE, DWORD) {
	var processId DWORD
	ret, _, _ := syscall.Syscall(uintptr(getWindowThreadProcessId), 2,
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processId)), 0)
	return HANDLE(ret), processId
}

func GetCurrentProcessId() DWORD {
	id, _, _ := syscall.Syscall(uintptr(getCurrentProcessId), 0, 0, 0, 0)
	return DWORD(id)
}

func ShowWindowAsync(hwnd HWND, cmdshow int) bool {
	ret, _, _ := syscall.Syscall(uintptr(showWindowAsync), 2,
		uintptr(hwnd),
		uintptr(cmdshow), 0)
	return ret != 0
}

func hideConsole() {
	console := GetConsoleWindow()
	if console != 0 {
		_, consoleProcID := GetWindowThreadProcessId(console)
		if GetCurrentProcessId() == consoleProcID {
			ShowWindowAsync(console, SW_HIDE)
		}
	}
}
