package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/smart-mcp-proxy/mcpproxy-go/internal/config"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/transport"
)

func newArgumentPrivacyUnitProxy(t *testing.T) *MCPProxyServer {
	t.Helper()
	p := createTestMCPProxyServer(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, p.upstreamManager.ShutdownAll(ctx))
		require.NoError(t, p.Close())
	})
	return p
}

// ARG-PXY-02's typed boundary is owned by MCPProxy. The installed upstream
// SDK does not preserve this type, code or data over Streamable HTTP.
func TestArgumentPrivacyTypedErrors(t *testing.T) {
	p := newArgumentPrivacyUnitProxy(t)
	marker := "typedM_雪/~"
	for _, indexed := range []bool{false, true} {
		if indexed {
			require.NoError(t, p.index.IndexTool(&config.ToolMetadata{
				Name: "private:check", ServerName: "private", ParamsJSON: argumentPrivacySchema,
			}))
		}
		for _, nestedHTTP := range []bool{false, true} {
			t.Run(fmt.Sprintf("indexed=%t/http=%t", indexed, nestedHTTP), func(t *testing.T) {
				original := &transport.JSONRPCError{
					Code: -32602, Message: "Invalid params: " + marker,
					Data: map[string]interface{}{marker: []interface{}{map[string]string{"path": "/" + marker, "value": marker}}},
				}
				if nestedHTTP {
					original.HTTPError = &transport.HTTPError{
						StatusCode: 400, Body: `{"rejected":"` + marker + `"}`,
						URL: "https://fixture.invalid/mcp", Method: "POST",
						Err: errors.New(marker),
					}
				}
				before, err := json.Marshal(original)
				require.NoError(t, err)
				wrapped := fmt.Errorf("outer request detail %s: %w", marker, original)
				result := p.createToolErrorResponse(wrapped, "private", "check", config.ToolArgumentErrorModeRedacted)
				body := decodeErrorBody(t, result)
				require.Equal(t, "Invalid tool arguments.", body["error"])
				privacyAssertNoMarkers(t, result, marker)
				if indexed {
					privacyAssertIndexedError(t, result, "private:check", true)
				} else {
					require.NotContains(t, body, "input_schema")
					require.NotContains(t, body, "hint")
					require.Equal(t, float64(-32602), body["error_code"])
					require.Contains(t, body, "error_data")
					require.Nil(t, body["error_data"])
					if nestedHTTP {
						require.Len(t, body, 5)
						httpDetails := body["http_details"].(map[string]interface{})
						require.Equal(t, float64(400), httpDetails["status_code"])
						require.Equal(t, "", httpDetails["response_body"])
						require.Equal(t, "https://fixture.invalid/mcp", httpDetails["server_url"])
					} else {
						require.Len(t, body, 3)
					}
				}
				for _, mode := range []string{"", config.ToolArgumentErrorModeDetailed} {
					detailed := p.createToolErrorResponse(wrapped, "private", "check", mode)
					require.Contains(t, fwdText(detailed), marker, "default/detailed retain original diagnosis")
					require.Equal(t, p.createDetailedErrorResponse(wrapped, "private", "check"), detailed)
				}
				after, err := json.Marshal(original)
				require.NoError(t, err)
				require.Equal(t, before, after, "redaction must not mutate shared message/data/nested HTTP errors")
				if nestedHTTP {
					require.Equal(t, marker, original.HTTPError.Err.Error())
				}
			})
		}
	}
}

func TestArgumentPrivacyClassificationControls(t *testing.T) {
	p := newArgumentPrivacyUnitProxy(t)
	marker := "controlM_雪/~"
	positive := errors.New("validation failed for argument: " + marker)
	for _, indexed := range []bool{false, true} {
		if indexed {
			require.NoError(t, p.index.IndexTool(&config.ToolMetadata{Name: "private:check", ServerName: "private", ParamsJSON: argumentPrivacySchema}))
		}
		t.Run(fmt.Sprintf("narrow_untyped/indexed=%t", indexed), func(t *testing.T) {
			result := p.createToolErrorResponse(positive, "private", "check", config.ToolArgumentErrorModeRedacted)
			body := decodeErrorBody(t, result)
			require.Equal(t, "Invalid tool arguments.", body["error"])
			privacyAssertNoMarkers(t, result, marker)
			if indexed {
				privacyAssertIndexedError(t, result, "private:check", true)
			} else {
				require.Len(t, body, 4)
				require.NotContains(t, body, "input_schema")
				require.NotContains(t, body, "hint")
				require.NotContains(t, body, "error_code", "an untyped error has no numeric code to invent")
			}
		})
	}
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"http_auth", &transport.HTTPError{StatusCode: 401, Body: "validation failed for argument: " + marker}},
		{"http_bad_request", &transport.HTTPError{StatusCode: 400, Body: "missing required property " + marker}},
		{"rpc_internal", &transport.JSONRPCError{Code: -32603, Message: "missing required field " + marker, Data: marker}},
		{"auth", errors.New("authentication failed; missing required field " + marker)},
		{"network", errors.New("connection refused; validation failed for argument " + marker)},
		{"timeout", errors.New("request timed out; validation failed for argument " + marker)},
		{"deadline", fmt.Errorf("%s: %w", marker, context.DeadlineExceeded)},
		{"internal", errors.New("internal failure: " + marker)},
		{"generic_invalid", errors.New("Invalid params: " + marker)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, eligible := classifyUpstreamInvalidParams(tc.err)
			require.False(t, eligible)
			detailed := p.createDetailedErrorResponse(tc.err, "private", "check")
			redacted := p.createToolErrorResponse(tc.err, "private", "check", config.ToolArgumentErrorModeRedacted)
			require.Equal(t, detailed, redacted, "noneligible errors retain their complete established response")
			require.NotContains(t, decodeErrorBody(t, redacted), "input_schema")
		})
	}
}
