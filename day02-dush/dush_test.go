package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestTopNKeepsBiggest(t *testing.T) {
	tt := &topN{cap: 3}
	for _, sz := range []int64{1, 5, 3, 4, 2} {
		tt.add(entry{"x", sz, false})
	}
	if tt.h.Len() != 3 {
		t.Fatalf("len = %d, want 3", tt.h.Len())
	}
	var got []int64
	for _, e := range tt.h {
		got = append(got, e.size)
	}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	want := []int64{3, 4, 5}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestWalkAggregates(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), 100)
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "sub", "b.txt"), 200)
	mustWrite(t, filepath.Join(root, "sub", "c.txt"), 300)

	tt := &topN{cap: 10}
	sem := make(chan struct{}, 4)
	total := walk(root, tt, sem)

	if total != 600 {
		t.Fatalf("total = %d, want 600", total)
	}

	found := false
	for _, e := range tt.h {
		if filepath.Base(e.path) == "sub" && e.dir && e.size == 500 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected sub/ at size 500 in heap, got %+v", tt.h)
	}
}

func TestRenderSkipsRootAndSortsDesc(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "big.bin"), 5000)
	mustWrite(t, filepath.Join(root, "small.txt"), 10)

	tt := &topN{cap: 10}
	sem := make(chan struct{}, 2)
	walk(root, tt, sem)

	var buf bytes.Buffer
	render(tt, root, 5, &buf)
	out := buf.String()

	// La racine elle-même ne doit jamais apparaître.
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.HasSuffix(line, "  "+root) || strings.HasSuffix(line, "  "+root+string(filepath.Separator)) {
			t.Fatalf("root path leaked into output:\n%s", out)
		}
	}
	if !strings.Contains(out, "big.bin") {
		t.Fatalf("output should mention big.bin:\n%s", out)
	}
	// Ordre décroissant : big.bin avant small.txt.
	if strings.Index(out, "big.bin") > strings.Index(out, "small.txt") {
		t.Fatalf("big.bin should appear before small.txt:\n%s", out)
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		0:       "0 B",
		100:     "100 B",
		1024:    "1.0 KiB",
		1536:    "1.5 KiB",
		1 << 20: "1.0 MiB",
		1 << 30: "1.0 GiB",
	}
	for n, want := range cases {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func mustWrite(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.WriteFile(path, make([]byte, size), 0644); err != nil {
		t.Fatal(err)
	}
}
