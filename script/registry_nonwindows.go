// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build !windows

package script

import (
	"github.com/gentee/gentee/core"
)

type RegKey struct {
	Existed bool
}

func RegistrySubkeys(root int64, subkey string, access int64) (*core.Array, error) {
	return nil, Unsupported(`RegistrySubkeys`)
}

func RegistryValues(root int64, subkey string, access int64) (*core.Array, error) {
	return nil, Unsupported(`RegistryValues`)
}

func CreateRegistryKey(root int64, subkey string, access int64) (*RegKey, error) {
	return nil, Unsupported(`CreateRegistryKey`)
}

func CloseRegistryKey(*RegKey) error {
	return Unsupported(`CloseRegistryKey`)
}

func SetRegistryValue(key *RegKey, name string, vtype int64, value string) error {
	return Unsupported(`SetRegistryValue`)
}

func DeleteRegistryKey(root int64, subkey string, access int64) error {
	return Unsupported(`DeleteRegistryKey`)
}

func DeleteRegistryValue(key *RegKey, name string) error {
	return Unsupported(`DeleteRegistryValue`)
}

func GetRegistryValue(key *RegKey, name string, def string) (string, error) {
	return ``, Unsupported(`GetRegistryValue`)
}

func OpenRegistryKey(root int64, subkey string, access int64) (*RegKey, error) {
	return nil, Unsupported(`OpenRegistryKey`)
}
