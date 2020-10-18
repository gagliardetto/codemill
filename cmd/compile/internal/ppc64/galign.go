// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ppc64

import (
	"github.com/gagliardetto/codemill/cmd/compile/internal/gc"
	"github.com/gagliardetto/codemill/cmd/internal/obj/ppc64"
	"github.com/gagliardetto/codemill/cmd/internal/objabi"
)

func Init(arch *gc.Arch) {
	arch.LinkArch = &ppc64.Linkppc64
	if objabi.GOARCH == "ppc64le" {
		arch.LinkArch = &ppc64.Linkppc64le
	}
	arch.REGSP = ppc64.REGSP
	arch.MAXWIDTH = 1 << 60

	arch.ZeroRange = zerorange
	arch.Ginsnop = ginsnop
	arch.Ginsnopdefer = ginsnopdefer

	arch.SSAMarkMoves = ssaMarkMoves
	arch.SSAGenValue = ssaGenValue
	arch.SSAGenBlock = ssaGenBlock
}
