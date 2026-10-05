package spectest

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/t14raptor/go-fwasm/runtime"
)

const suite = "testdata/wg-2.0"

func TestSpec(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(suite, "*.json"))
	if err != nil || len(files) == 0 {
		t.Skipf("no spec tests in %s; run ./fetch.sh", suite)
	}
	total := Counts{}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		t.Run(name, func(t *testing.T) {
			failures, err := RunScript(f, total)
			if err != nil {
				t.Fatal(err)
			}
			for _, fail := range failures {
				t.Error(fail)
			}
		})
	}

	kinds := make([]string, 0, len(total))
	for k := range total {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		c := total[k]
		t.Logf("%-24s pass %6d  fail %4d  skip %4d", k, c[Pass], c[Fail], c[Skip])
	}
}

// TestSpecHooks runs the suite with hooks installed, which runs every
// function as compiled, without inlined calls, and through the general
// call path.
func TestSpecHooks(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(suite, "*.json"))
	if err != nil || len(files) == 0 {
		t.Skipf("no spec tests in %s; run ./fetch.sh", suite)
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		t.Run(name, func(t *testing.T) {
			failures, err := runScript(f, Counts{}, &runtime.Hooks{})
			if err != nil {
				t.Fatal(err)
			}
			for _, fail := range failures {
				t.Error(fail)
			}
		})
	}
}
