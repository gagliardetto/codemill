// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package arm

import (
	"github.com/gagliardetto/codemill/cmd/compile/internal/gc"
	"github.com/gagliardetto/codemill/cmd/compile/internal/ssa"
	"github.com/gagliardetto/codemill/cmd/internal/obj/arm"
	"github.com/gagliardetto/codemill/cmd/internal/objabi"
)

func Init(arch *gc.Arch) {
	arch.LinkArch = &arm.Linkarm
	arch.REGSP = arm.REGSP
	arch.MAXWIDTH = (1 << 32) - 1
	arch.SoftFloat = objabi.GOARM == 5
	arch.ZeroRange = zerorange
	arch.Ginsnop = ginsnop
	arch.Ginsnopdefer = ginsnop

	arch.SSAMarkMoves = func(s *gc.SSAGenState, b *ssa.Block) {}
	arch.SSAGenValue = ssaGenValue
	arch.SSAGenBlock = ssaGenBlock
}
