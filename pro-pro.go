// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by Eonza license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"eonza/lib"
	"eonza/users"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	TrialDisabled = -1
	TrialOff      = 0
	TrialOn       = 1
)

type CallbackFunc func() error

var (
	Active              = true
	CallbackPassCounter CallbackFunc
	CallbackTitle       func() string
	CallbackTrial       func() int
	CallbackTaskCheck   func(uint32, uint32) (bool, error)
)

func IsTwofa() bool {
	return Active && proStorage.Settings.Twofa
}

func TwofaQR(id uint32) (ret string, err error) {
	if !IsTwofa() {
		return
	}
	var (
		key *otp.Key
		img image.Image
	)
	proMutex.Lock()
	defer proMutex.Unlock()
	user, ok := proStorage.Users[id]
	if !ok {
		return
	}
	if _, ok = proStorage.Twofa[id]; ok {
		//key, err = otp.NewKeyFromURL(kurl)
		return
	}

	key, err = totp.Generate(totp.GenerateOpts{
		Issuer:      CallbackTitle(),
		AccountName: user.Nickname,
		Secret:      []byte(lib.UniqueName(16)),
	})
	proStorage.Twofa[id] = key.URL()
	if err = ProSaveStorage(false); err != nil {
		return
	}
	if img, err = key.Image(200, 200); err != nil {
		return
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	ret = `data:image/png;base64,` + base64.StdEncoding.EncodeToString(buf.Bytes())
	return
}

func ValidateOTP(user users.User, passcode string) error {
	kurl, ok := proStorage.Twofa[user.ID]
	if !ok || len(kurl) == 0 {
		return fmt.Errorf(`There is not OTP secret key`)
	}
	key, err := otp.NewKeyFromURL(kurl)
	if err != nil {
		return nil
	}
	if !totp.Validate(passcode, key.Secret()) {
		return fmt.Errorf(`Invalid one-time passcode`)
	}
	return nil
}

func Licensed() bool {
	return true
}

func SetActive() {
	Active = true
}

func LoadPro(psw []byte, counter uint32, cfgpath string, userspath string) {
	roles, ulist := users.InitUsers(psw, counter)
	ProLoadStorage(cfgpath)
	proStorage.Roles[users.XAdminID] = roles[users.XAdminID]
	proStorage.Roles[users.TimersID] = roles[users.TimersID]
	proStorage.Roles[users.EventsID] = roles[users.EventsID]
	proStorage.Users[users.XRootID] = ulist[users.XRootID]
	LoadUsers(userspath)
}

func SecureConstants() map[string]string {
	return secureConst
}

func IsAutoFill() bool {
	return Active && proStorage.Settings.AutoFill
}
