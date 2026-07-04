#include "textflag.h"

// func callSetVolumeScalar(fn uintptr, aev uintptr, levelBits uint32, eventContextGUID uintptr) uintptr
TEXT ·callSetVolumeScalar(SB), NOSPLIT, $40-32
	MOVQ fn+0(FP), AX
	MOVQ aev+8(FP), CX
	
	MOVSS levelBits+16(FP), X1
	MOVL levelBits+16(FP), DX
	
	MOVQ eventContextGUID+24(FP), R8

	CALL AX

	MOVQ AX, ret+32(FP)
	RET
