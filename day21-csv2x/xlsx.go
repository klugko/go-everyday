package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// Un .xlsx n'est qu'une archive zip contenant quelques fichiers XML au format
// Office Open XML. On en écrit le strict minimum pour qu'Excel et LibreOffice
// ouvrent le classeur : pas de styles, pas de sharedStrings — les chaînes sont
// posées « inline » dans la feuille, ce qui fait un fichier de moins à gérer.

// Parties statiques de l'archive : elles ne dépendent jamais des données.
const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`

const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

const workbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets>
</workbook>`

const workbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`

func writeXLSX(w io.Writer, header []string, rows [][]string) error {
	z := zip.NewWriter(w)

	parts := []struct{ name, body string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", rootRels},
		{"xl/workbook.xml", workbook},
		{"xl/_rels/workbook.xml.rels", workbookRels},
		{"xl/worksheets/sheet1.xml", sheet(header, rows)},
	}
	for _, p := range parts {
		f, err := z.Create(p.name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(f, p.body); err != nil {
			return err
		}
	}
	return z.Close()
}

// sheet construit la feuille : l'en-tête puis les lignes de données.
func sheet(header []string, rows [][]string) string {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)

	row := 1
	writeRow(&b, row, header)
	for _, r := range rows {
		row++
		// On aligne chaque ligne sur la largeur de l'en-tête : Excel n'aime pas
		// les références de cellule qui sautent une colonne.
		cells := make([]string, len(header))
		for i := range header {
			cells[i] = cell(r, i)
		}
		writeRow(&b, row, cells)
	}

	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func writeRow(b *bytes.Buffer, row int, cells []string) {
	fmt.Fprintf(b, `<row r="%d">`, row)
	for col, val := range cells {
		ref := fmt.Sprintf("%s%d", colName(col), row)
		if looksNumeric(val) {
			// Pas d'attribut de type = nombre : Excel le traite comme tel.
			fmt.Fprintf(b, `<c r="%s"><v>%s</v></c>`, ref, val)
		} else {
			fmt.Fprintf(b, `<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, escapeXML(val))
		}
	}
	b.WriteString(`</row>`)
}

// colName traduit un indice de colonne (0-based) en lettres Excel : 0→A, 25→Z,
// 26→AA. C'est de la numération biais-26, d'où le n-- avant chaque chiffre.
func colName(n int) string {
	name := ""
	for n >= 0 {
		name = string(rune('A'+n%26)) + name
		n = n/26 - 1
	}
	return name
}

func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
