package shared

import "encoding/json"

// JSONOrNull marshals v for storage in a MySQL JSON column. MySQL's JSON type
// rejects an empty string with error 3140 ("Invalid JSON text") — the Go zero
// value for string is "", so any AuditLog.BeforeData/AfterData left unset must
// go through this helper (which returns the JSON literal "null") rather than
// being written as an empty string.
func JSONOrNull(v interface{}) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 {
		return "null"
	}
	return string(b)
}
