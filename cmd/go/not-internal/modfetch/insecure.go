// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package modfetch

import (
	"github.com/gagliardetto/codemill/cmd/go/not-internal/cfg"
	"github.com/gagliardetto/codemill/cmd/go/not-internal/get"
	"github.com/gagliardetto/codemill/cmd/go/not-internal/str"
)

// allowInsecure reports whether we are allowed to fetch this path in an insecure manner.
func allowInsecure(path string) bool {
	return get.Insecure || str.GlobsMatchPath(cfg.GOINSECURE, path)
}
