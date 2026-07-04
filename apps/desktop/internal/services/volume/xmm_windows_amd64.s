#include "textflag.h"

// func callSetVolumeScalar(fn uintptr, aev uintptr, level float32, eventContextGUID uintptr) uintptr
TEXT ·callSetVolumeScalar(SB), NOSPLIT, $0
	MOVQ fn+0(FP), AX
	MOVQ aev+8(FP), CX
	MOVSS level+16(FP), X1
	MOVQ eventContextGUID+24(FP), R8

	SUBQ $40, SP
	CALL AX
	ADDQ $40, SP

	MOVQ AX, ret+32(FP)
	RET
