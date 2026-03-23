package shared

import (
	"os"
)

// ResetTierCacheForTest routes tier-cache reads and writes to an isolated temp dir for tests.
func ResetTierCacheForTest() {
	tierCacheDirOverrideMu.Lock()
	override := tierCacheDirOverride
	if override == "" {
		tempDir, err := os.MkdirTemp("", "asc-tier-cache-*")
		if err != nil {
			tierCacheDirOverrideMu.Unlock()
			return
		}
		override = tempDir
		tierCacheDirOverride = override
	}
	tierCacheDirOverrideMu.Unlock()

	_ = os.RemoveAll(override)
	_ = os.MkdirAll(override, 0o755)
}

func resetTierCacheDirOverrideForTest() {
	tierCacheDirOverrideMu.Lock()
	override := tierCacheDirOverride
	tierCacheDirOverride = ""
	tierCacheDirOverrideMu.Unlock()

	if override == "" {
		return
	}
	_ = os.RemoveAll(override)
}
