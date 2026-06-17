package main

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

// jeu de données partagé : une colonne texte, une numérique, un code à zéro
// de tête (qui doit rester du texte), une cellule vide.
var (
	header = []string{"name", "age", "zip", "note"}
	rows   = [][]string{
		{"Alice", "30", "007", "ok"},
		{"Bob", "25", "75001", ""},
	}
)

func TestJSONInfersTypes(t *testing.T) {
	var b bytes.Buffer
	if err := writeJSON(&b, header, rows); err != nil {
		t.Fatal(err)
	}
	out := b.String()

	// 30 est un nombre nu, "007" garde ses guillemets, la note vide devient null.
	for _, want := range []string{`"name": "Alice"`, `"age": 30`, `"zip": "007"`, `"note": null`} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON ne contient pas %q\n%s", want, out)
		}
	}
}

func TestMarkdownTable(t *testing.T) {
	var b bytes.Buffer
	if err := writeMarkdown(&b, header, rows); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")

	want := []string{
		"| name | age | zip | note |",
		"| --- | --- | --- | --- |",
		"| Alice | 30 | 007 | ok |",
		"| Bob | 25 | 75001 |  |",
	}
	for i, w := range want {
		if i >= len(lines) || lines[i] != w {
			t.Errorf("ligne %d = %q, attendu %q", i, get(lines, i), w)
		}
	}
}

func TestMarkdownEscapesPipes(t *testing.T) {
	var b bytes.Buffer
	writeMarkdown(&b, []string{"a"}, [][]string{{"x|y"}})
	if !strings.Contains(b.String(), `x\|y`) {
		t.Errorf("le pipe n'a pas été échappé :\n%s", b.String())
	}
}

func TestXLSXIsValidZip(t *testing.T) {
	var b bytes.Buffer
	if err := writeXLSX(&b, header, rows); err != nil {
		t.Fatal(err)
	}

	r, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		t.Fatalf("archive illisible : %v", err)
	}

	// Toutes les parties obligatoires doivent être présentes.
	files := map[string]string{}
	for _, f := range r.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(data)
	}
	for _, must := range []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/worksheets/sheet1.xml"} {
		if _, ok := files[must]; !ok {
			t.Errorf("partie manquante : %s", must)
		}
	}

	sheet := files["xl/worksheets/sheet1.xml"]
	// Alice est en A2 (chaîne inline), 30 en B2 (nombre nu).
	if !strings.Contains(sheet, `<c r="A2" t="inlineStr"><is><t xml:space="preserve">Alice</t></is></c>`) {
		t.Errorf("cellule texte mal écrite :\n%s", sheet)
	}
	if !strings.Contains(sheet, `<c r="B2"><v>30</v></c>`) {
		t.Errorf("cellule numérique mal écrite :\n%s", sheet)
	}
}

func TestColName(t *testing.T) {
	cases := map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB", 51: "AZ", 52: "BA", 701: "ZZ", 702: "AAA"}
	for n, want := range cases {
		if got := colName(n); got != want {
			t.Errorf("colName(%d) = %q, attendu %q", n, got, want)
		}
	}
}

func TestLooksNumeric(t *testing.T) {
	yes := []string{"0", "30", "-12", "3.14", "1e3"}
	no := []string{"", "007", "+5", "75001abc", "12,5"}
	for _, s := range yes {
		if !looksNumeric(s) {
			t.Errorf("%q devrait être numérique", s)
		}
	}
	for _, s := range no {
		if looksNumeric(s) {
			t.Errorf("%q ne devrait pas être numérique", s)
		}
	}
}

func TestPickFormat(t *testing.T) {
	cases := []struct{ f, out, want string }{
		{"", "", "json"},
		{"json", "", "json"},
		{"", "data.md", "md"},
		{"markdown", "", "md"},
		{"", "report.xlsx", "xlsx"},
		{"excel", "", "xlsx"},
		{"yaml", "", ""},
	}
	for _, c := range cases {
		if got := pickFormat(c.f, c.out); got != c.want {
			t.Errorf("pickFormat(%q,%q) = %q, attendu %q", c.f, c.out, got, c.want)
		}
	}
}

func get(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return "<absent>"
}
