// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !go1.18
// +build !go1.18

package singleflight // import "golang.org/x/sync/singleflight/v2"

// singleflight/v2 requires Go 1.18 or later for generics support. To avoid a
// confusing "build constraints exclude all Go files in" compile error on Go
// 1.17 and earlier, we add this file and the below code. The code will fail to
// compile on Go 1.17 or earlier and should help folks understand what they need
// to do.

const versionRequired = REQUIRES_GO_1_18_OR_LATER
