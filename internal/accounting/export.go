package accounting

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var printTmpl = template.Must(template.New("print").Parse(`<!DOCTYPE html>
<html lang="ru"><head>
<meta charset="UTF-8">
<title>{{.Title}}</title>
<style>
body{font-family:Arial,sans-serif;font-size:12px;margin:20px}
h2{margin-bottom:4px}
p.meta{color:#555;margin:0 0 12px}
table{border-collapse:collapse;width:100%}
th,td{border:1px solid #ccc;padding:5px 8px;text-align:left;white-space:nowrap}
th{background:#f0f0f0;font-weight:bold}
tfoot td{font-weight:bold;background:#f9f9f9}
@media print{@page{margin:10mm}button{display:none}}
</style>
</head><body>
<h2>{{.Title}}</h2>
<p class="meta">{{.Meta}}</p>
<table>
<thead><tr>{{range .Headers}}<th>{{.}}</th>{{end}}</tr></thead>
<tbody>{{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>{{end}}</tbody>
{{if .Footer}}<tfoot><tr>{{range .Footer}}<td>{{.}}</td>{{end}}</tr></tfoot>{{end}}
</table>
</body></html>`))

type exportData struct {
	Title   string
	Meta    string
	Headers []string
	Rows    [][]string
	Footer  []string
}

func buildExcel(title string, headers []string, rows [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	header, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8E8E8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// Title row
	f.SetCellValue(sheet, "A1", title)
	f.SetCellStyle(sheet, "A1", "A1", bold)

	// Header row
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, header)
	}

	// Data rows
	for rowIdx, row := range rows {
		for col, val := range row {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx+3)
			f.SetCellValue(sheet, cell, val)
		}
	}

	// Auto-width
	for col := range headers {
		colName, _ := excelize.ColumnNumberToName(col + 1)
		f.SetColWidth(sheet, colName, colName, 18)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildHTML(data exportData) ([]byte, error) {
	var buf bytes.Buffer
	if err := printTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildTSV(headers []string, rows [][]string) []byte {
	var sb strings.Builder
	sb.WriteString(strings.Join(headers, "\t"))
	sb.WriteByte('\n')
	for _, row := range rows {
		sb.WriteString(strings.Join(row, "\t"))
		sb.WriteByte('\n')
	}
	return []byte(sb.String())
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02.01.2006")
}

func formatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02.01.2006 15:04")
}

func contentType(format string) string {
	switch format {
	case "excel":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "html":
		return "text/html; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

func fileExt(format string) string {
	switch format {
	case "excel":
		return "xlsx"
	case "html":
		return "html"
	default:
		return "tsv"
	}
}
