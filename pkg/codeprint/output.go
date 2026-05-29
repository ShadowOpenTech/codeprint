package codeprint

// ProducerName is the canonical producer name in the Meta envelope.
const ProducerName = "codeprint"

// NewOutput wraps a fingerprint in the output envelope. schemaURL may be empty
// (pre-1.0). generatedAt is an RFC3339 string supplied by the caller and lives
// only in Meta — never in the fingerprint — so the fingerprint stays
// reproducible. Pass an empty generatedAt to omit it.
func NewOutput(fp *Fingerprint, schemaURL, version, generatedAt string) Output {
	return Output{
		Schema: schemaURL,
		Meta: Meta{
			SchemaVersion: SchemaVersion,
			Producer:      Producer{Name: ProducerName, Version: version},
			GeneratedAt:   generatedAt,
		},
		Fingerprint: *fp,
	}
}
