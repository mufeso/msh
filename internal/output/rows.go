package output

import (
	"encoding/csv"
	"io"
)

// WriteRows writes tabular data in format f: an aligned table (text),
// CSV with a header row, or the items as JSON. headers and rows feed the
// text and CSV forms; items feed the JSON form.
//
// T is a type parameter, so callers pass their own slice type
// ([]inventory.Server, []taskSummary, ...) without converting it.
func WriteRows[T any](w io.Writer, f Format, headers []string, rows [][]string, items []T) error {
	switch f {
	case JSON:
		if items == nil {
			items = []T{} // [] instead of null
		}
		return WriteJSON(w, items)
	case CSV:
		cw := csv.NewWriter(w)
		if err := cw.Write(headers); err != nil {
			return err
		}
		return cw.WriteAll(rows) // WriteAll also flushes
	default:
		return Table(w, headers, rows)
	}
}
