package radiusctl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

var stdout = os.Stdout

// Writer is a simple column-aligned table writer wrapping tabwriter.
type Writer struct {
	w *tabwriter.Writer
}

// NewWriter returns a Writer configured for borderless aligned output
// with 2-space minimum spacing between columns.
func NewWriter() *Writer {
	return &Writer{
		w: tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0),
	}
}

// Row writes a tab-separated row to the table.
func (t *Writer) Row(cols ...string) {
	fmt.Fprintln(t.w, strings.Join(cols, "\t"))
}

// Flush flushes the table output.
func (t *Writer) Flush() { t.w.Flush() }

// PrintJSON marshals v to indented JSON and prints it. Used when --json is
// set. Does nothing itself if marshaling fails.
func PrintJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	fmt.Fprintln(stdout, string(b))
}

// KV prints key-value pairs row-by-row. Useful for single-object display
// (status, balance, cleanup results).
func KV(pairs ...[2]string) {
	w := NewWriter()
	for _, p := range pairs {
		w.Row(p[0]+":", p[1])
	}
	w.Flush()
}

// humanBytes formats a byte count into a human-readable string (KB/MB/GB).
func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// yesNo formats a bool as "yes" / "no".
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// itoa and uitoa are thin wrappers to keep call sites readable without
// importing strconv everywhere.
func itoa(n int) string     { return fmt.Sprintf("%d", n) }
func uitoa(n uint64) string { return fmt.Sprintf("%d", n) }

// ptrStrOr returns *s when set, else fallback (used for nil-or-empty bools).
func strOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
