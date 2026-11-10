package mcpserver_test

import (
	"encoding/json"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// jsonOf decodes the JSON of the answer of a tool into the given value.
//
// The answer of a tool is a JSON document: the test decodes it and binds
// every field to its own key, so a money that travels under the key of
// another one cannot hide in the text of the answer.
func jsonOf(t *testing.T, result *sdk.CallToolResult, into any) {
	t.Helper()

	text := textOf(t, result)

	if err := json.Unmarshal([]byte(text), into); err != nil {
		t.Fatalf("the answer of the tool is not a JSON document: %v\n%s", err, text)
	}
}
