package chartmux

import _ "embed"

const SchemaURL = "https://chartmux.dev/schema/v1.json"

//go:embed schema/v1.json
var schemaV1 []byte

func SchemaJSON() []byte {
	return append([]byte(nil), schemaV1...)
}
