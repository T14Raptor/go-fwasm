package replay

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/t14raptor/go-fwasm/runtime"
)

// TestReplay replays recordings (see the package doc for the format):
//
//	FWASM_REPLAY_WASM=module.wasm FWASM_REPLAY_LOGS='rec-*.json' go test ./replay
//
// FWASM_REPLAY_MEMORY names the exported memory (default "memory"), and
// FWASM_REPLAY_HASH_EVERY samples the memory checks at export returns.
func TestReplay(t *testing.T) {
	m, paths := recordings(t)
	opts := Options{Memory: os.Getenv("FWASM_REPLAY_MEMORY"), HashEvery: 1}
	if n, err := strconv.Atoi(os.Getenv("FWASM_REPLAY_HASH_EVERY")); err == nil {
		opts.HashEvery = n
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			log, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			stats, err := Run(m, log, opts)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%d events: %d export calls, %d import calls, %d memory hashes match (%v)",
				len(log.Events), stats.Calls, stats.Imports, stats.Hashes, time.Since(start))
		})
	}
}

// BenchmarkReplay times the last call in each recording, after replaying
// the calls before it untimed on a fresh instance. Memory is checked once,
// at the end:
//
//	FWASM_REPLAY_WASM=module.wasm FWASM_REPLAY_LOGS='rec-*.json' go test ./replay -run '^$' -bench Replay
func BenchmarkReplay(b *testing.B) {
	m, paths := recordings(b)
	opts := Options{Memory: os.Getenv("FWASM_REPLAY_MEMORY"), FinalHashOnly: true}
	for _, path := range paths {
		log, err := Load(path)
		if err != nil {
			b.Fatal(err)
		}
		from := 0
		if len(log.Marks) > 0 {
			from = log.Marks[len(log.Marks)-1]
		}
		b.Run(filepath.Base(path), func(b *testing.B) {
			b.SetBytes(int64(len(log.Input)))
			var calls int
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				r, err := New(m, log, opts)
				if err == nil {
					err = r.RunTo(from)
				}
				if err != nil {
					b.Fatal(err)
				}
				before := r.Stats().Calls
				b.StartTimer()
				if err := r.RunTo(len(log.Events)); err != nil {
					b.Fatal(err)
				}
				calls = r.Stats().Calls - before
			}
			b.ReportMetric(float64(calls), "calls/op")
		})
	}
}

// recordings compiles FWASM_REPLAY_WASM and expands FWASM_REPLAY_LOGS, a
// comma-separated list of globs.
func recordings(tb testing.TB) (*runtime.Module, []string) {
	tb.Helper()
	wasmPath, logs := os.Getenv("FWASM_REPLAY_WASM"), os.Getenv("FWASM_REPLAY_LOGS")
	if wasmPath == "" || logs == "" {
		tb.Skip("set FWASM_REPLAY_WASM and FWASM_REPLAY_LOGS")
	}
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		tb.Fatal(err)
	}
	m, err := runtime.Compile(wasm)
	if err != nil {
		tb.Fatal(err)
	}
	var paths []string
	for _, pattern := range strings.Split(logs, ",") {
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			tb.Fatalf("no recordings match %q", pattern)
		}
		paths = append(paths, matches...)
	}
	return m, paths
}
