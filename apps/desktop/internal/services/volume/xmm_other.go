//go:build !amd64 || !windows

package volume

func callSetVolumeScalar(fn uintptr, aev uintptr, level float32, eventContextGUID uintptr) uintptr {
	// No-op fallback for non-Windows/non-AMD64 platforms.
	return 0
}
