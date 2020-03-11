// Copyright 2020 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package lib

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
)

// AppendLeft is filling slice at the left
func AppendLeft(in []byte) []byte {
	if len(in) >= 32 {
		return in
	}
	return append(make([]byte, 32-len(in)), in...)
}

// GenerateKeys generates a random pair of ECDSA private and public binary keys
func GenerateKeys() ([]byte, []byte, error) {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return private.D.Bytes(), append(AppendLeft(private.PublicKey.X.Bytes()),
		AppendLeft(private.PublicKey.Y.Bytes())...), nil
}
