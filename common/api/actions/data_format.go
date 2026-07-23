package actions

// DataFormat is the format of structured data to search in when resolving a LookupKey.
//
// +kubebuilder:validation:Enum=resource_path;json;sql
type DataFormat string

const (
	// FormatResourcePath matches a path segment after LookupKey and captures the next segment.
	// Example (lookupKey = "orders"):
	//   text:  /api/v1/orders/abc-123
	//   regex: (?:^|/)orders/([^/\s"?&#]+)
	//   capture: abc-123
	FormatResourcePath DataFormat = "resource_path"

	// FormatJSON matches a JSON key equal to LookupKey and captures its value.
	// Example (lookupKey = "user_id"):
	//   text:  {"user_id": "abc123", "name": "foo"}
	//   regex: (?:^|[\s,{])"?user_id"?\s*:\s*"?([^"\s,}\]]+)
	//   capture: abc123
	FormatJSON DataFormat = "json"

	// FormatSQL matches a SQL column equal to LookupKey and captures the compared value.
	// Example (lookupKey = "user_id"):
	//   text:  WHERE user_id = '42' AND status = 'ok'
	//   regex: (?:^|[\s,(])user_id\s*=\s*'?([^'\s,;)]+)
	//   capture: 42
	FormatSQL DataFormat = "sql"
)
