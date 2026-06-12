package main

import (
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	cases := []struct {
		name string
		md   string
		want string
	}{
		{"titre h1", "# Bonjour", "<h1>Bonjour</h1>\n"},
		{"titre h3 avec inline", "### Du *style*", "<h3>Du <em>style</em></h3>\n"},
		{"sept dièses = paragraphe", "####### trop", "<p>####### trop</p>\n"},
		{"dièse collé = paragraphe", "#pas-un-titre", "<p>#pas-un-titre</p>\n"},
		{"gras et italique", "du **gras** et de la *pente*",
			"<p>du <strong>gras</strong> et de la <em>pente</em></p>\n"},
		{"apostrophe échappée", "l'outil",
			"<p>l&#39;outil</p>\n"},
		{"lien", "voir [le repo](https://example.com)",
			"<p>voir <a href=\"https://example.com\">le repo</a></p>\n"},
		{"code inline échappé", "la balise `<b>` en HTML",
			"<p>la balise <code>&lt;b&gt;</code> en HTML</p>\n"},
		{"pas de gras dans le code", "`**pas gras**`",
			"<p><code>**pas gras**</code></p>\n"},
		{"backtick orphelin", "un ` tout seul", "<p>un ` tout seul</p>\n"},
		{"liste à puces", "- un\n- deux",
			"<ul>\n<li>un</li>\n<li>deux</li>\n</ul>\n"},
		{"liste numérotée", "1. un\n2. deux",
			"<ol>\n<li>un</li>\n<li>deux</li>\n</ol>\n"},
		{"citation sur deux lignes", "> première\n> seconde",
			"<blockquote><p>première seconde</p></blockquote>\n"},
		{"séparateur", "---", "<hr>\n"},
		{"paragraphe sur deux lignes", "une ligne\nla suite",
			"<p>une ligne la suite</p>\n"},
		{"deux paragraphes", "premier\n\nsecond",
			"<p>premier</p>\n<p>second</p>\n"},
		{"bloc de code échappé", "```\nif a < b {\n```",
			"<pre><code>if a &lt; b {\n</code></pre>\n"},
		{"bloc de code avec langage", "```go\nx := 1\n```",
			"<pre><code class=\"language-go\">x := 1\n</code></pre>\n"},
		{"bloc de code jamais fermé", "```\nseul",
			"<pre><code>seul\n</code></pre>\n"},
		{"crlf toléré", "# Windows\r\n\r\ntexte",
			"<h1>Windows</h1>\n<p>texte</p>\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := convert(c.md); got != c.want {
				t.Errorf("convert(%q)\n  got  %q\n  want %q", c.md, got, c.want)
			}
		})
	}
}

func TestTitle(t *testing.T) {
	if got := title("intro\n\n# Le vrai titre\n", "doc.md"); got != "Le vrai titre" {
		t.Errorf("titre h1 attendu, reçu %q", got)
	}
	if got := title("pas de titre ici", "notes/reunion.md"); got != "reunion" {
		t.Errorf("nom de fichier attendu, reçu %q", got)
	}
	if got := title("rien", ""); got != "Document" {
		t.Errorf("libellé par défaut attendu, reçu %q", got)
	}
}

func TestPage(t *testing.T) {
	doc := page("Mon <titre>", "<p>corps</p>\n")
	for _, want := range []string{
		"<!doctype html>",
		"<title>Mon &lt;titre&gt;</title>",
		"<style>",
		"<p>corps</p>",
		"charset=\"utf-8\"",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("la page devrait contenir %q", want)
		}
	}
}
