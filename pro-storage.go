package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"sort"
	"sync"

	"eonza/lib"
	"eonza/users"

	"github.com/gentee/gentee/vm"
	"github.com/kataras/golog"
	"gopkg.in/yaml.v3"
)

const (
	ProExt = `pro`
)

/*type User struct {
	User users.User
}*/

// Storage contains all application data
type ProStorage struct {
	Settings users.ProSettings
	License  LicenseInfo
	Roles    map[uint32]users.Role
	Users    map[uint32]users.User
	Twofa    map[uint32]string
	Secure   []byte
}

type Secure struct {
	ID    uint32 `json:"id" yaml:"id"`
	Desc  string `json:"desc" yaml:"desc"`
	Value string `json:"value" yaml:"value"`
}

var (
	secure      map[string]Secure
	secureConst map[string]string
	passphrase  []byte
	cfgname     string
	proStorage  = ProStorage{
		Settings: users.ProSettings{
			AutoFill: true,
		},
		Roles: make(map[uint32]users.Role),
		Users: make(map[uint32]users.User),
		Twofa: make(map[uint32]string),
	}
	proMutex       = &sync.Mutex{}
	ErrUnknownUser = fmt.Errorf(`Unknown user`)
)

func ProLoadStorage(cfgpath string) {
	cfgname = lib.ChangeExt(cfgpath, ProExt)
	if _, err := os.Stat(cfgname); os.IsNotExist(err) {
		return
	}
	data, err := os.ReadFile(cfgname)
	if err != nil {
		golog.Fatal(err)
	}
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	if err = dec.Decode(&proStorage); err != nil {
		golog.Fatal(err)
	}
	for key, role := range proStorage.Roles {
		proStorage.Roles[key] = users.ParseAllow(role)
	}
}

func ProSaveStorage(encrypt bool) error {
	var (
		data           bytes.Buffer
		err            error
		ret, encrypted []byte
	)
	if encrypt {
		if len(secure) > 0 {
			if len(passphrase) == 0 {
				return fmt.Errorf(`Undefined Master Password`)
			}
			if ret, err = yaml.Marshal(secure); err != nil {
				return err
			}
			if encrypted, err = vm.AESEncrypt(passphrase, ret); err != nil {
				return err
			}
		}
		proStorage.Secure = encrypted
	}
	enc := gob.NewEncoder(&data)
	if err = enc.Encode(proStorage); err != nil {
		return err
	}
	return os.WriteFile(cfgname, data.Bytes(), 0777 /*os.ModePerm*/)
}

func GetRole(id uint32) (role users.Role, ok bool) {
	if !Active && id != users.XAdminID {
		return
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	role, ok = proStorage.Roles[id]
	return
}

func GetUser(id uint32) (user users.User, ok bool) {
	if !Active && id != users.XRootID {
		return
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	user, ok = proStorage.Users[id]
	return
}

func ProGetUserRole(id uint32) (uname string, rname string) {
	// ? proMutex.Lock()
	// ? defer proMutex.Unlock()
	if user, ok := proStorage.Users[id]; ok {
		uname = user.Nickname
		if role, ok := proStorage.Roles[user.RoleID]; ok {
			rname = role.Name
		}
	}
	return
}

func GetUsers() []users.User {
	proMutex.Lock()
	defer proMutex.Unlock()
	ret := make([]users.User, 0)
	for _, user := range proStorage.Users {
		ret = append(ret, user)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Nickname < ret[j].Nickname
	})
	return ret
}

func IncPassCounter(id uint32) error {
	var (
		user users.User
		ok   bool
	)
	if user, ok = GetUser(id); !ok {
		return ErrUnknownUser
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	user.PassCounter++
	proStorage.Users[id] = user
	return ProSaveStorage(false)
}

func SetUserPassword(id uint32, hash []byte) error {
	var (
		user users.User
		ok   bool
	)
	if user, ok = GetUser(id); !ok {
		return ErrUnknownUser
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	if len(hash) == 0 && Active && len(proStorage.Users) > 1 {
		return fmt.Errorf(`Empty password`)
	}
	user.PassCounter++
	user.PasswordHash = hash
	proStorage.Users[id] = user
	return ProSaveStorage(false)
}

func ProSettings() users.ProSettings {
	return proStorage.Settings
}

func IsDecrypted() bool {
	return secure != nil || len(proStorage.Settings.Master) == 0
}
