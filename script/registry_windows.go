// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

// +build windows

package script

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gentee/gentee/core"
	"golang.org/x/sys/windows/registry"
)

type RegKey struct {
	Key     registry.Key
	Existed bool
}

func RootKey(root int64) registry.Key {
	switch root {
	case 1:
		return registry.CURRENT_USER
	case 2:
		return registry.LOCAL_MACHINE
	case 3:
		return registry.USERS
	case 4:
		return registry.CURRENT_CONFIG
	case 5:
		return registry.PERFORMANCE_DATA
	}
	return registry.CLASSES_ROOT
}

func RegistrySubkeys(root int64, subkey string, access int64) (*core.Array, error) {
	var list []string
	k, err := registry.OpenKey(RootKey(root), subkey, registry.ENUMERATE_SUB_KEYS|uint32(access))
	if err != nil {
		return nil, err
	}
	if list, err = k.ReadSubKeyNames(0); err != nil {
		return nil, err
	}
	ret := core.NewArray()
	ret.Data = make([]interface{}, len(list))
	for i, item := range list {
		ret.Data[i] = item
	}
	k.Close()
	return ret, nil
}

func CreateRegistryKey(root int64, subkey string, access int64) (*RegKey, error) {
	k, exist, err := registry.CreateKey(RootKey(root), subkey, registry.ALL_ACCESS|uint32(access))
	if err != nil {
		return nil, err
	}
	return &RegKey{
		Key:     k,
		Existed: exist,
	}, nil
}

func CloseRegistryKey(key *RegKey) error {
	return key.Key.Close()
}

func SetRegistryValue(key *RegKey, name string, vtype int64, value string) error {
	switch vtype {
	case registry.SZ:
		return key.Key.SetStringValue(name, value)
	case registry.EXPAND_SZ:
		return key.Key.SetExpandStringValue(name, value)
	case registry.DWORD:
		ival, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return err
		}
		return key.Key.SetDWordValue(name, uint32(ival))
	}
	return fmt.Errorf("Unsupported Registry type %d", vtype)
}

func RegistryValues(root int64, subkey string, access int64) (*core.Array, error) {
	var list []string
	k, err := registry.OpenKey(RootKey(root), subkey, registry.READ|uint32(access))
	if err != nil {
		return nil, err
	}
	if list, err = k.ReadValueNames(0); err != nil {
		return nil, err
	}
	ret := core.NewArray()
	ret.Data = make([]interface{}, len(list))
	for i, item := range list {
		ret.Data[i] = item
	}
	k.Close()
	return ret, nil
}

func DeleteRegistryKey(root int64, subkey string, access int64) error {
	var (
		key string
		err error
		k   registry.Key
	)
	subkey = strings.Trim(subkey, `\`)
	off := strings.LastIndexByte(subkey, '\\')
	if off > 0 {
		key = subkey[off+1:]
		k, err = registry.OpenKey(RootKey(root), subkey[:off], registry.ALL_ACCESS|uint32(access))
		if err != nil {
			return err
		}
		defer k.Close()
	} else {
		key = subkey
		k = RootKey(root)
	}
	return registry.DeleteKey(k, key)
}

func DeleteRegistryValue(key *RegKey, name string) error {
	return key.Key.DeleteValue(name)
}

func GetRegistryValue(key *RegKey, name string, def string) (ret string, err error) {
	var (
		vtype uint32
		iret  uint64
	)
	ret, vtype, err = key.Key.GetStringValue(name)
	if err != nil {
		if err == registry.ErrUnexpectedType {
			if vtype == registry.DWORD || vtype == registry.QWORD {
				if iret, _, err = key.Key.GetIntegerValue(name); err == nil {
					ret = strconv.FormatUint(iret, 10)
				}
			}
		}
		if err == registry.ErrNotExist && len(def) > 0 {
			err = nil
			ret = def
		}
	}
	return
}

func OpenRegistryKey(root int64, subkey string, access int64) (*RegKey, error) {
	k, err := registry.OpenKey(RootKey(root), subkey, registry.READ|uint32(access))
	if err != nil {
		return nil, err
	}
	return &RegKey{
		Key: k,
	}, nil
}
