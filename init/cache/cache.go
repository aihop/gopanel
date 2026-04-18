package cache

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/init/cache/badger_db"
	"github.com/dgraph-io/badger/v4"
	"github.com/shirou/gopsutil/v4/mem"
)

func Init() {
	c := global.CONF.System.Cache
	_ = os.RemoveAll(c)
	_ = os.Mkdir(c, 0755)
	options := badger.Options{
		Dir:                c,
		ValueDir:           c,
		ValueLogFileSize:   64 << 20,
		ValueLogMaxEntries: 10 << 20,
		VLogPercentile:     0.1,

		MemTableSize:                  32 << 20,
		BaseTableSize:                 2 << 20,
		BaseLevelSize:                 10 << 20,
		TableSizeMultiplier:           2,
		LevelSizeMultiplier:           10,
		MaxLevels:                     7,
		NumGoroutines:                 4,
		MetricsEnabled:                true,
		NumCompactors:                 2,
		NumLevelZeroTables:            5,
		NumLevelZeroTablesStall:       15,
		NumMemtables:                  1,
		BloomFalsePositive:            0.01,
		BlockSize:                     2 * 1024,
		SyncWrites:                    false,
		NumVersionsToKeep:             1,
		CompactL0OnClose:              false,
		VerifyValueChecksum:           false,
		BlockCacheSize:                32 << 20,
		IndexCacheSize:                0,
		ZSTDCompressionLevel:          1,
		EncryptionKey:                 []byte{},
		EncryptionKeyRotationDuration: 10 * 24 * time.Hour, // Default 10 days.
		DetectConflicts:               true,
		NamespaceOffset:               -1,
	}

	profile := strings.ToLower(strings.TrimSpace(os.Getenv("GOPANEL_CACHE_PROFILE")))
	if profile == "" || profile == "auto" {
		if v, ok := readEnvBool("GOPANEL_CACHE_AUTO"); !ok || v {
			profile = autoProfileByMemory(profile)
		}
	}
	switch profile {
	case "low":
		options.MemTableSize = 8 << 20
		options.BlockCacheSize = 8 << 20
		options.ValueLogFileSize = 16 << 20
		options.MetricsEnabled = false
		options.NumCompactors = 1
		options.NumGoroutines = 2
	case "tiny":
		options.MemTableSize = 4 << 20
		options.BlockCacheSize = 4 << 20
		options.ValueLogFileSize = 8 << 20
		options.MetricsEnabled = false
		options.NumCompactors = 1
		options.NumGoroutines = 1
	}

	if mb, ok := readEnvInt("GOPANEL_CACHE_MEMTABLE_MB"); ok {
		options.MemTableSize = int64(clampInt(mb, 1, 256)) << 20
	}
	if mb, ok := readEnvInt("GOPANEL_CACHE_BLOCKCACHE_MB"); ok {
		options.BlockCacheSize = int64(clampInt(mb, 0, 512)) << 20
	}
	if mb, ok := readEnvInt("GOPANEL_CACHE_VLOGFILE_MB"); ok {
		options.ValueLogFileSize = int64(clampInt(mb, 1, 256)) << 20
	}
	if v, ok := readEnvBool("GOPANEL_CACHE_METRICS"); ok {
		options.MetricsEnabled = v
	}
	if n, ok := readEnvInt("GOPANEL_CACHE_COMPACTORS"); ok {
		options.NumCompactors = clampInt(n, 1, 8)
	}
	if n, ok := readEnvInt("GOPANEL_CACHE_GOROUTINES"); ok {
		options.NumGoroutines = clampInt(n, 1, 16)
	}

	cache, err := badger.Open(options)
	if err != nil {
		panic(err)
	}
	_ = cache.DropAll()
	global.CacheDb = cache
	global.CACHE = badger_db.NewCacheDB(cache)
	global.LOG.Info("init cache successfully")
}

func readEnvInt(key string) (int, bool) {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}

func readEnvBool(key string) (bool, bool) {
	s := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if s == "" {
		return false, false
	}
	switch s {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func autoProfileByMemory(profile string) string {
	vm, err := mem.VirtualMemory()
	if err != nil || vm == nil || vm.Total == 0 {
		return profile
	}
	mb := vm.Total / (1024 * 1024)
	if mb <= 1024 {
		return "tiny"
	}
	if mb <= 2048 {
		return "low"
	}
	return profile
}
