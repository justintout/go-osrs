package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
)

// table aligns text columns. Every caller writes to a strings.Builder,
// which cannot fail, so write errors are ignored.
type table struct{ w *tabwriter.Writer }

func newTable(w io.Writer, headers ...string) table {
	t := table{tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)}
	if len(headers) > 0 {
		t.row(headers...)
	}
	return t
}

func (t table) row(cells ...string) { fmt.Fprintln(t.w, strings.Join(cells, "\t")) }
func (t table) flush()              { _ = t.w.Flush() }

// num formats n with comma separators.
func num(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func rank(r int) string {
	if r < 0 {
		return "-"
	}
	return num(int64(r))
}

func need(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func signed(n int) string {
	if n > 0 {
		return "+" + num(int64(n))
	}
	return num(int64(n))
}
