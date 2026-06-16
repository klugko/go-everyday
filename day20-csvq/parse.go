package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Plutôt qu'un vrai analyseur SQL, on tokenise une fois pour toutes puis on
// consomme les jetons clause par clause. Suffisant pour SELECT/WHERE/GROUP BY,
// et bien plus lisible qu'un découpage à coups de strings.Split.

type token struct {
	s      string
	quoted bool 

func (t *token) is(kw string) bool {
	return t != nil && !t.quoted && strings.EqualFold(t.s, kw)
}

// tokenize découpe la requête : mots, nombres, chaînes 'quotées', ponctuation
// et opérateurs de comparaison, en ignorant les espaces.
func tokenize(q string) ([]token, error) {
	var toks []token
	r := []rune(q)
	for i := 0; i < len(r); {
		c := r[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '\'':
			i++
			var sb strings.Builder
			for i < len(r) && r[i] != '\'' {
				sb.WriteRune(r[i])
				i++
			}
			if i >= len(r) {
				return nil, fmt.Errorf("apostrophe non fermée")
			}
			i++ // apostrophe de fermeture.
			toks = append(toks, token{sb.String(), true})
		case strings.ContainsRune("(),*", c):
			toks = append(toks, token{string(c), false})
			i++
		case strings.ContainsRune("=<>!", c):
			op := string(c)
			i++
			// Deuxième caractère des opérateurs à deux signes : <=, >=, !=, <>.
			if i < len(r) && (r[i] == '=' || (c == '<' && r[i] == '>')) {
				op += string(r[i])
				i++
			}
			toks = append(toks, token{op, false})
		default:
			j := i
			for j < len(r) && !unicode.IsSpace(r[j]) && !strings.ContainsRune("(),*=<>!'", r[j]) {
				j++
			}
			toks = append(toks, token{string(r[i:j]), false})
			i = j
		}
	}
	return toks, nil
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() *token {
	if p.pos < len(p.toks) {
		return &p.toks[p.pos]
	}
	return nil
}

func (p *parser) next() *token {
	t := p.peek()
	if t != nil {
		p.pos++
	}
	return t
}

func parse(q string) (*query, error) {
	toks, err := tokenize(q)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	out := &query{limit: -1}

	if !p.next().is("SELECT") {
		return nil, fmt.Errorf("la requête doit commencer par SELECT")
	}
	if out.cols, err = p.parseSelectors(); err != nil {
		return nil, err
	}

	if !p.next().is("FROM") {
		return nil, fmt.Errorf("clause FROM attendue")
	}
	f := p.next()
	if f == nil {
		return nil, fmt.Errorf("nom de fichier attendu après FROM")
	}
	out.from = f.s

	// Les clauses qui suivent sont toutes optionnelles ; on les prend dans
	// l'ordre où elles viennent plutôt que d'imposer une séquence rigide.
	for {
		t := p.peek()
		switch {
		case t == nil:
			return out, nil
		case t.is("WHERE"):
			p.next()
			if out.where, err = p.parseWhere(); err != nil {
				return nil, err
			}
		case t.is("GROUP"):
			p.next()
			if !p.next().is("BY") {
				return nil, fmt.Errorf("GROUP doit être suivi de BY")
			}
			out.groupBy = p.parseNameList()
		case t.is("ORDER"):
			p.next()
			if !p.next().is("BY") {
				return nil, fmt.Errorf("ORDER doit être suivi de BY")
			}
			out.orderBy = p.parseOrder()
		case t.is("LIMIT"):
			p.next()
			n := p.next()
			if n == nil {
				return nil, fmt.Errorf("nombre attendu après LIMIT")
			}
			if out.limit, err = strconv.Atoi(n.s); err != nil || out.limit < 0 {
				return nil, fmt.Errorf("LIMIT invalide : %s", n.s)
			}
		default:
			return nil, fmt.Errorf("jeton inattendu : %q", t.s)
		}
	}
}

func (p *parser) parseSelectors() ([]selector, error) {
	var cols []selector
	for {
		s, err := p.parseSelector()
		if err != nil {
			return nil, err
		}
		cols = append(cols, s)
		if p.peek().is(",") {
			p.next()
			continue
		}
		return cols, nil
	}
}

var aggregates = map[string]bool{"count": true, "sum": true, "avg": true, "min": true, "max": true}

func (p *parser) parseSelector() (selector, error) {
	t := p.next()
	if t == nil {
		return selector{}, fmt.Errorf("colonne attendue dans le SELECT")
	}
	var s selector
	switch {
	case t.is("*"):
		s.col = "*"
	case aggregates[strings.ToLower(t.s)] && p.peek().is("("):
		s.fn = strings.ToLower(t.s)
		p.next() 
		arg := p.next()
		if arg == nil {
			return s, fmt.Errorf("argument attendu pour %s()", s.fn)
		}
		s.col = arg.s
		if !p.next().is(")") {
			return s, fmt.Errorf("parenthèse fermante attendue après %s(", s.fn)
		}
	default:
		s.col = t.s
	}
	if p.peek().is("AS") {
		p.next()
		a := p.next()
		if a == nil {
			return s, fmt.Errorf("alias attendu après AS")
		}
		s.alias = a.s
	}
	return s, nil
}

// parseWhere produit une forme normale disjonctive : le ET lie plus fort que le
// OU (a AND b OR c → (a ET b) OU c), comme en SQL.
func (p *parser) parseWhere() ([][]cond, error) {
	var groups [][]cond
	var cur []cond
	for {
		c, err := p.parseCond()
		if err != nil {
			return nil, err
		}
		cur = append(cur, c)
		switch {
		case p.peek().is("AND"):
			p.next()
		case p.peek().is("OR"):
			p.next()
			groups = append(groups, cur)
			cur = nil
		default:
			return append(groups, cur), nil
		}
	}
}

func (p *parser) parseCond() (cond, error) {
	col := p.next()
	if col == nil {
		return cond{}, fmt.Errorf("condition attendue après WHERE")
	}
	opTok := p.next()
	if opTok == nil {
		return cond{}, fmt.Errorf("opérateur attendu après %s", col.s)
	}
	op := normalizeOp(opTok)
	if op == "" {
		return cond{}, fmt.Errorf("opérateur inconnu : %s", opTok.s)
	}
	val := p.next()
	if val == nil {
		return cond{}, fmt.Errorf("valeur attendue après %s %s", col.s, op)
	}
	return cond{col: col.s, op: op, val: val.s}, nil
}

func normalizeOp(t *token) string {
	if t.is("LIKE") {
		return "LIKE"
	}
	switch t.s {
	case "=", "==":
		return "="
	case "!=", "<>":
		return "!="
	case "<", "<=", ">", ">=":
		return t.s
	}
	return ""
}

func (p *parser) parseNameList() []string {
	var names []string
	for {
		t := p.next()
		if t == nil {
			return names
		}
		names = append(names, t.s)
		if p.peek().is(",") {
			p.next()
			continue
		}
		return names
	}
}

func (p *parser) parseOrder() []orderKey {
	var keys []orderKey
	for {
		t := p.next()
		if t == nil {
			return keys
		}
		k := orderKey{col: t.s}
		switch {
		case p.peek().is("ASC"):
			p.next()
		case p.peek().is("DESC"):
			p.next()
			k.desc = true
		}
		keys = append(keys, k)
		if p.peek().is(",") {
			p.next()
			continue
		}
		return keys
	}
}
