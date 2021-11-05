// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package main

func Migrate() {
	// TODO: update all installed extensions with newer version
	// ...

	storage.Version = GetVersion()
}
