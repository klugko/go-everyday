package main

import (
	"strings"
	"testing"
)

// Le round-trip est le vrai contrat : convertir puis reconvertir doit
// redonner exactement la source (aux espaces près que json.MarshalIndent fixe).
func TestRoundTrip(t *testing.T) {
	srcJSON := `{
  "name": "fmtj",
  "nested": {
    "count": 3,
    "ok": true
  },
  "tags": [
    "json",
    "yaml"
  ]
}
`
	yamlOut, err := convert([]byte(srcJSON), "json", "yaml")
	if err != nil {
		t.Fatalf("json→yaml : %v", err)
	}
	back, err := convert(yamlOut, "yaml", "json")
	if err != nil {
		t.Fatalf("yaml→json : %v", err)
	}
	if string(back) != srcJSON {
		t.Errorf("round-trip cassé :\nattendu :\n%s\nobtenu :\n%s", srcJSON, back)
	}
}

func TestJSONToYAML(t *testing.T) {
	out, err := convert([]byte(`{"a":1,"b":["x","z"]}`), "json", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{"a: 1", "b:", "- x", "- z"} {
		if !strings.Contains(got, want) {
			t.Errorf("sortie YAML ne contient pas %q :\n%s", want, got)
		}
	}
}

func TestYAMLToJSON(t *testing.T) {
	in := "name: bob\nage: 42\n"
	out, err := convert([]byte(in), "yaml", "json")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, `"name": "bob"`) || !strings.Contains(got, `"age": 42`) {
		t.Errorf("YAML→JSON inattendu :\n%s", got)
	}
}

func TestInvalidInput(t *testing.T) {
	if _, err := convert([]byte(`{bad`), "json", "yaml"); err == nil {
		t.Error("JSON invalide aurait dû échouer")
	}
}

func TestUnknownFormat(t *testing.T) {
	if _, err := convert([]byte(`{}`), "toml", "yaml"); err == nil {
		t.Error("format d'entrée inconnu aurait dû échouer")
	}
}

func TestDetectFormat(t *testing.T) {
	cases := map[string]string{
		"x.json": "json",
		"x.yaml": "yaml",
		"x.yml":  "yaml",
		"X.JSON": "json",
		"x.toml": "",
		"x":      "",
	}
	for name, want := range cases {
		if got := detectFormat(name); got != want {
			t.Errorf("detectFormat(%q) = %q, attendu %q", name, got, want)
		}
	}
}

func TestOpposite(t *testing.T) {
	if opposite("json") != "yaml" || opposite("yaml") != "json" {
		t.Error("opposite ne bascule pas correctement")
	}
}
