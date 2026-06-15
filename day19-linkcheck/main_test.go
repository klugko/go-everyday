package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLinksFindsInlineAndImageLinks(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	md := "# Doc\n\nvoir [le guide](guide.md) et ![logo](img/logo.png).\n\n" +
		"```\n[pas un lien](nope.md)\n```\n\n[externe](https://example.com)\n"
	if err := os.WriteFile(f, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := links(f)
	if err != nil {
		t.Fatal(err)
	}
	var urls []string
	for _, l := range got {
		urls = append(urls, l.url)
	}
	// le lien dans le bloc de code ne doit pas apparaître.
	want := []string{"guide.md", "img/logo.png", "https://example.com"}
	if !reflect.DeepEqual(urls, want) {
		t.Errorf("liens = %v, attendu %v", urls, want)
	}
}

func TestLinksDropsTitle(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	os.WriteFile(f, []byte("[x](page.md \"un titre\")\n"), 0o644)

	got, _ := links(f)
	if len(got) != 1 || got[0].url != "page.md" {
		t.Errorf("le titre devrait être retiré, reçu %+v", got)
	}
}

func TestCheckLocalResolvesRelativeToFile(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "doc.md")
	os.WriteFile(filepath.Join(dir, "exists.md"), []byte("ok"), 0o644)

	if r := checkLocal(from, "exists.md#section"); r != "" {
		t.Errorf("cible présente, devrait tenir, reçu %q", r)
	}
	if r := checkLocal(from, "missing.md"); r == "" {
		t.Error("cible absente, devrait être signalée morte")
	}
	if r := checkLocal(from, "#anchor"); r != "" {
		t.Errorf("ancre pure, rien à ouvrir, reçu %q", r)
	}
}

func TestCheckOneLeavesSchemesAlone(t *testing.T) {
	for _, u := range []string{"#top", "mailto:a@b.c", "tel:0102030405"} {
		if r := checkOne(nil, link{url: u}); r != "" {
			t.Errorf("%q ne devrait pas être jugé, reçu %q", u, r)
		}
	}
}

func TestCheckHTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dead" {
			http.Error(w, "nope", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if r := checkHTTP(srv.Client(), srv.URL+"/ok"); r != "" {
		t.Errorf("200 devrait passer, reçu %q", r)
	}
	if r := checkHTTP(srv.Client(), srv.URL+"/dead"); r == "" {
		t.Error("404 devrait être signalé")
	}
}

func TestCheckReportsDeadSorted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	from := filepath.Join(dir, "a.md")
	os.WriteFile(filepath.Join(dir, "here.md"), []byte("x"), 0o644)

	in := []link{
		{from, 5, "missing.md"}, 
		{from, 2, srv.URL},      
		{from, 9, "here.md"},   
	}
	dead := check(in, srv.Client(), 4)
	if len(dead) != 2 {
		t.Fatalf("2 morts attendus, %d : %+v", len(dead), dead)
	}
	// triés par ligne : le lien externe (2) avant le fichier manquant (5).
	if dead[0].link.line != 2 || dead[1].link.line != 5 {
		t.Errorf("résultats mal triés : %+v", dead)
	}
}
