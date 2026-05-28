package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestScanGroupsBySize(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a"), []byte("hello"))
	mustWrite(t, filepath.Join(root, "b"), []byte("world"))
	mustWrite(t, filepath.Join(root, "c"), []byte("differentsize"))
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "sub", "d"), []byte("yummy"))
	// Fichier vide : doit être ignoré.
	mustWrite(t, filepath.Join(root, "empty"), nil)

	bySize := scan(root)
	// 5 octets : a, b, d. 13 octets : c. Le vide est sauté.
	if got := len(bySize[5]); got != 3 {
		t.Fatalf("taille 5 : %d entrées, attendu 3", got)
	}
	if got := len(bySize[13]); got != 1 {
		t.Fatalf("taille 13 : %d entrées, attendu 1", got)
	}
	if _, ok := bySize[0]; ok {
		t.Fatalf("le fichier vide ne devrait pas apparaître")
	}
}

func TestHashFileMatchesContent(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "x")
	mustWrite(t, p, []byte("hello"))
	// SHA-256("hello").
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	got, err := hashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("hash = %s, attendu %s", got, want)
	}
}

func TestGroupFindsDuplicates(t *testing.T) {
	root := t.TempDir()
	// Trois copies du même contenu, plus un fichier de même taille mais
	// contenu différent (faux positif côté taille).
	mustWrite(t, filepath.Join(root, "a.txt"), []byte("twins!"))
	mustWrite(t, filepath.Join(root, "b.txt"), []byte("twins!"))
	mustWrite(t, filepath.Join(root, "c.txt"), []byte("twins!"))
	mustWrite(t, filepath.Join(root, "imposter"), []byte("FAKE!!"))
	// Un solo de taille unique : jamais hashé.
	mustWrite(t, filepath.Join(root, "solo"), []byte("xxxxxxxxxxx"))

	groups := group(scan(root), runtime.NumCPU())
	if len(groups) != 1 {
		t.Fatalf("%d groupes, attendu 1 (%v)", len(groups), groups)
	}
	for _, g := range groups {
		if len(g.paths) != 3 {
			t.Fatalf("groupe : %d fichiers, attendu 3", len(g.paths))
		}
		if g.size != 6 {
			t.Fatalf("taille du groupe : %d, attendu 6", g.size)
		}
		// L'imposteur ne doit jamais atterrir dans un groupe de doublons.
		for _, p := range g.paths {
			if strings.HasSuffix(p, "imposter") {
				t.Fatalf("imposteur dans le groupe : %v", g.paths)
			}
		}
	}
}

func TestGroupIgnoresUniqueSizes(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a"), []byte("aaaa"))
	mustWrite(t, filepath.Join(root, "b"), []byte("bb"))
	mustWrite(t, filepath.Join(root, "c"), []byte("ccc"))
	if got := len(group(scan(root), 2)); got != 0 {
		t.Fatalf("%d groupes, attendu 0", got)
	}
}

func TestSortedGroupsByWasteDesc(t *testing.T) {
	groups := map[string]*dupGroup{
		"h1": {size: 100, paths: []string{"a", "b"}},               // 100 récup
		"h2": {size: 10, paths: []string{"c", "d", "e", "f"}},      // 30 récup
		"h3": {size: 1000, paths: []string{"g", "h"}},              // 1000 récup
	}
	list := sortedGroups(groups)
	sizes := []int64{list[0].size, list[1].size, list[2].size}
	want := []int64{1000, 100, 10}
	for i, w := range want {
		if sizes[i] != w {
			t.Fatalf("ordre = %v, attendu %v", sizes, want)
		}
	}
}

func TestRenderEmpty(t *testing.T) {
	var buf bytes.Buffer
	render(nil, "/x", &buf)
	if !strings.Contains(buf.String(), "Pas de doublons") {
		t.Fatalf("sortie inattendue : %q", buf.String())
	}
}

func TestRenderShowsGroupsAndTotal(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a"), []byte("dup"))
	mustWrite(t, filepath.Join(root, "b"), []byte("dup"))

	groups := group(scan(root), 2)
	list := sortedGroups(groups)
	var buf bytes.Buffer
	render(list, root, &buf)
	out := buf.String()
	if !strings.Contains(out, "Total") {
		t.Fatalf("le total manque : %q", out)
	}
	// Les chemins doivent apparaître en relatif (pas de chemin absolu).
	if strings.Contains(out, root) {
		t.Fatalf("chemin absolu dans la sortie : %q", out)
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		0:       "0 B",
		512:     "512 B",
		1024:    "1.0 KiB",
		1 << 20: "1.0 MiB",
		1 << 30: "1.0 GiB",
	}
	keys := make([]int64, 0, len(cases))
	for k := range cases {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		if got := humanSize(k); got != cases[k] {
			t.Errorf("humanSize(%d) = %q, attendu %q", k, got, cases[k])
		}
	}
}

func TestPromptKeepsChosenDeletesRest(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	c := filepath.Join(root, "c")
	mustWrite(t, a, []byte("same"))
	mustWrite(t, b, []byte("same"))
	mustWrite(t, c, []byte("same"))

	list := sortedGroups(group(scan(root), 2))
	if len(list) != 1 || len(list[0].paths) != 3 {
		t.Fatalf("setup raté : %v", list)
	}
	// On garde le numéro 2 — donc 1 et 3 doivent disparaître.
	keep := list[0].paths[1]
	gone1, gone3 := list[0].paths[0], list[0].paths[2]

	var out bytes.Buffer
	prompt(list, root, strings.NewReader("2\n"), &out)

	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("le fichier gardé a disparu : %v", err)
	}
	if _, err := os.Stat(gone1); !os.IsNotExist(err) {
		t.Fatalf("le fichier 1 aurait dû être supprimé (%v)", err)
	}
	if _, err := os.Stat(gone3); !os.IsNotExist(err) {
		t.Fatalf("le fichier 3 aurait dû être supprimé (%v)", err)
	}
}

func TestPromptQuitStopsImmediately(t *testing.T) {
	root := t.TempDir()
	// Deux groupes pour vérifier que "q" coupe avant le 2e.
	mustWrite(t, filepath.Join(root, "a1"), []byte("groupA"))
	mustWrite(t, filepath.Join(root, "a2"), []byte("groupA"))
	mustWrite(t, filepath.Join(root, "b1"), []byte("groupBBBB"))
	mustWrite(t, filepath.Join(root, "b2"), []byte("groupBBBB"))

	list := sortedGroups(group(scan(root), 2))
	if len(list) != 2 {
		t.Fatalf("attendu 2 groupes, got %d", len(list))
	}

	var out bytes.Buffer
	prompt(list, root, strings.NewReader("q\n"), &out)

	// Rien ne doit avoir été supprimé.
	for _, name := range []string{"a1", "a2", "b1", "b2"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("%s aurait dû être intact : %v", name, err)
		}
	}
}

func TestPromptEmptySkipsGroup(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a"), []byte("xx"))
	mustWrite(t, filepath.Join(root, "b"), []byte("xx"))

	list := sortedGroups(group(scan(root), 2))
	var out bytes.Buffer
	prompt(list, root, strings.NewReader("\n"), &out)

	for _, name := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("%s aurait dû survivre : %v", name, err)
		}
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
