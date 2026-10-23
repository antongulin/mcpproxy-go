package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/config"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/contracts"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/httpapi"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/management"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/oauth"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/runtime"
	"github.com/smart-mcp-proxy/mcpproxy-go/internal/runtime/stateview"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func argumentModeRequest(args map[string]interface{}) mcp.CallToolRequest {
	request := mcp.CallToolRequest{}
	request.Params.Arguments = args
	return request
}

// ADMIN-MODE-01: exercise the shared public management schema and the real
// add/update/patch/list dispatcher against temporary BBolt state.
func TestArgumentErrorModeMCPManagement(t *testing.T) {
	p := createTestMCPProxyServer(t)
	found := false
	for _, tool := range p.buildManagementTools() {
		if tool.Tool.Name != "upstream_servers" {
			continue
		}
		found = true
		property, ok := tool.Tool.InputSchema.Properties["tool_argument_error_mode"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "string", property["type"])
		require.ElementsMatch(t, []string{"", "detailed", "redacted"}, property["enum"])
		require.NotContains(t, tool.Tool.InputSchema.Required, "tool_argument_error_mode")
	}
	require.True(t, found)
	call := func(args map[string]interface{}, wantError bool) *mcp.CallToolResult {
		result, err := p.handleUpstreamServers(context.Background(), argumentModeRequest(args))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, wantError, result.IsError, fwdResultText(t, result))
		return result
	}
	control := &config.ServerConfig{Name: "control", Enabled: true, Headers: map[string]string{"Authorization": "synthetic-control"}}
	require.NoError(t, p.storage.SaveUpstreamServer(control))
	beforeControl, err := p.storage.GetUpstreamServer("control")
	require.NoError(t, err)
	call(map[string]interface{}{"operation": "add", "name": "private-upstream", "url": "http://127.0.0.1:1/mcp", "protocol": "streamable-http", "enabled": false, "tool_argument_error_mode": "redacted", "headers_json": `{"Authorization":"synthetic-private-token"}`, "oauth_json": `{"client_id":"synthetic-client","client_secret":"synthetic-private-secret"}`}, false)
	for _, operation := range []string{"update", "patch"} {
		for _, tc := range []struct{ mode, want string }{{"redacted", "redacted"}, {"omitted", "redacted"}, {"detailed", "detailed"}, {"redacted", "redacted"}} {
			args := map[string]interface{}{"operation": operation, "name": "private-upstream"}
			if tc.mode != "omitted" {
				args["tool_argument_error_mode"] = tc.mode
			}
			call(args, false)
			got, err := p.storage.GetUpstreamServer("private-upstream")
			require.NoError(t, err)
			require.Equal(t, tc.want, got.ToolArgumentErrorMode)
			require.False(t, got.Enabled)
			require.Equal(t, "synthetic-private-token", got.Headers["Authorization"])
			require.Equal(t, "synthetic-private-secret", got.OAuth.ClientSecret)
		}
	}
	before, err := p.storage.GetUpstreamServer("private-upstream")
	require.NoError(t, err)
	for _, operation := range []string{"add", "update", "patch"} {
		for _, invalid := range []interface{}{"unknown", "Redacted", 42, nil} {
			name := "private-upstream"
			if operation == "add" {
				name = "invalid-upstream"
			}
			call(map[string]interface{}{"operation": operation, "name": name, "url": "http://127.0.0.1:1/mcp", "enabled": false, "tool_argument_error_mode": invalid}, true)
		}
	}
	after, err := p.storage.GetUpstreamServer("private-upstream")
	require.NoError(t, err)
	require.Equal(t, before, after)
	_, err = p.storage.GetUpstreamServer("invalid-upstream")
	require.Error(t, err)
	listed := fwdResultText(t, call(map[string]interface{}{"operation": "list"}, false))
	require.Contains(t, listed, `"tool_argument_error_mode":"redacted"`)
	require.NotContains(t, listed, "synthetic-private-token")
	require.NotContains(t, listed, "synthetic-private-secret")
	afterControl, err := p.storage.GetUpstreamServer("control")
	require.NoError(t, err)
	require.Equal(t, beforeControl, afterControl)
}

// ADMIN-MODE-01 (refined): exercise existing REST POST/PATCH/GET routes with
// the real core controller, runtime, management service, and temporary DB/file.
// Seed status snapshots from persisted records at the component boundary;
// the supervisor is not started and no external connections are needed.
func TestArgumentErrorModeRESTManagement(t *testing.T) {
	p, rt := createTestProxyWithRuntimeCfg(t, nil, func(cfg *config.Config) {
		cfg.APIKey = "synthetic-admin-key"
		cfg.AllowServerAdd = true
		cfg.Tokenizer = &config.TokenizerConfig{Enabled: false}
	})
	main := p.mainServer
	main.logger = zap.NewNop()
	main.serverCtx = rt.AppContext()
	cfg := rt.ConfigSnapshot().Clone()
	path := config.GetConfigPath(cfg.DataDir)
	require.NoError(t, config.SaveConfig(cfg, path))
	rt.UpdateConfig(cfg, path)
	rt.SetManagementService(management.NewService(rt, cfg, path, nil, nil, zap.NewNop().Sugar()))
	api := httpapi.NewServer(main, zap.NewNop().Sugar(), nil)
	events := rt.SubscribeEvents()
	t.Cleanup(func() { rt.UnsubscribeEvents(events) })
	waitForChange := func() {
		t.Helper()
		deadline := time.NewTimer(10 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case evt := <-events:
				if evt.Type == runtime.EventTypeServersChanged && evt.Payload["reason"] == "tools_indexed" {
					return
				}
			case <-deadline.C:
				t.Fatal("server-change work did not finish")
			}
		}
	}
	request := func(method, url, body, key string, status int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, url, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-API-Key", key)
		w := httptest.NewRecorder()
		api.ServeHTTP(w, r)
		require.Equal(t, status, w.Code, w.Body.String())
		return w
	}
	control := &config.ServerConfig{Name: "control", Enabled: true, URL: "http://127.0.0.1:1/control", Protocol: "streamable-http", Headers: map[string]string{"Authorization": "synthetic-control"}}
	require.NoError(t, rt.StorageManager().SaveUpstreamServer(control))
	require.NoError(t, rt.SaveConfiguration())
	controlBefore, err := rt.StorageManager().GetUpstreamServer("control")
	require.NoError(t, err)
	request(http.MethodPost, "/api/v1/servers", `{"name":"unauthorized","url":"http://127.0.0.1:1/mcp","tool_argument_error_mode":"redacted"}`, "wrong-key", http.StatusUnauthorized)
	request(http.MethodPost, "/api/v1/servers", `{"name":"private-upstream","url":"http://127.0.0.1:1/mcp","protocol":"streamable-http","enabled":false,"tool_argument_error_mode":"redacted","headers":{"Authorization":"synthetic-private-token"}}`, cfg.APIKey, http.StatusOK)
	waitForChange()
	verify := func(want string) {
		t.Helper()
		servers, err := rt.StorageManager().ListUpstreamServers()
		require.NoError(t, err)
		for _, sc := range servers {
			rt.Supervisor().StateView().UpdateServer(sc.Name, func(status *stateview.ServerStatus) {
				status.Name, status.Config, status.Enabled, status.Quarantined = sc.Name, sc, sc.Enabled, sc.Quarantined
			})
		}
		w := request(http.MethodGet, "/api/v1/servers", "", cfg.APIKey, http.StatusOK)
		var response struct {
			Data contracts.GetServersResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		found := false
		for _, sc := range response.Data.Servers {
			if sc.Name == "private-upstream" {
				found = true
				require.Equal(t, want, sc.ToolArgumentErrorMode)
				require.False(t, sc.Enabled)
				require.NotEmpty(t, sc.Headers["Authorization"])
				require.NotEqual(t, "synthetic-private-token", sc.Headers["Authorization"])
			}
		}
		require.True(t, found, w.Body.String())
		require.NotContains(t, w.Body.String(), "synthetic-private-token")
		stored, err := rt.StorageManager().GetUpstreamServer("private-upstream")
		require.NoError(t, err)
		require.Equal(t, want, stored.ToolArgumentErrorMode)
		require.Equal(t, "synthetic-private-token", stored.Headers["Authorization"])
		disk, err := config.LoadFromFile(path)
		require.NoError(t, err)
		for _, sc := range disk.Servers {
			if sc.Name == "private-upstream" {
				require.Equal(t, want, sc.ToolArgumentErrorMode)
			}
		}
		// The actual fallback controller projects the same stored mode.
		service := rt.GetManagementService()
		rt.SetManagementService(nil)
		fallback := request(http.MethodGet, "/api/v1/servers", "", cfg.APIKey, http.StatusOK)
		rt.SetManagementService(service)
		var fallbackResponse struct {
			Data contracts.GetServersResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(fallback.Body.Bytes(), &fallbackResponse))
		fallbackFound := false
		for _, sc := range fallbackResponse.Data.Servers {
			if sc.Name == "private-upstream" {
				fallbackFound = true
				require.Equal(t, want, sc.ToolArgumentErrorMode)
			}
		}
		require.True(t, fallbackFound)
	}
	verify("redacted")
	for _, tc := range []struct{ body, want string }{{`{"args":["unrelated-change"]}`, "redacted"}, {`{"tool_argument_error_mode":"detailed"}`, "detailed"}, {`{"tool_argument_error_mode":"redacted"}`, "redacted"}} {
		request(http.MethodPatch, "/api/v1/servers/private-upstream/", tc.body, cfg.APIKey, http.StatusOK)
		waitForChange()
		verify(tc.want)
	}
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	storedBefore, err := rt.StorageManager().GetUpstreamServer("private-upstream")
	require.NoError(t, err)
	for _, body := range []string{`{"tool_argument_error_mode":"unknown"}`, `{"tool_argument_error_mode":42}`} {
		request(http.MethodPatch, "/api/v1/servers/private-upstream/", body, cfg.APIKey, http.StatusBadRequest)
	}
	request(http.MethodPost, "/api/v1/servers", `{"name":"invalid","url":"http://127.0.0.1:1/mcp","tool_argument_error_mode":"unknown"}`, cfg.APIKey, http.StatusBadRequest)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
	storedAfter, err := rt.StorageManager().GetUpstreamServer("private-upstream")
	require.NoError(t, err)
	require.Equal(t, storedBefore, storedAfter)
	_, err = rt.StorageManager().GetUpstreamServer("invalid")
	require.Error(t, err)
	afterControl, err := rt.StorageManager().GetUpstreamServer("control")
	require.NoError(t, err)
	require.Equal(t, controlBefore, afterControl)
	// The shared OAuth config views must retain the enum while masking secrets.
	view := oauth.RedactedServerConfigView(storedAfter, oauth.LiveRedaction)
	require.Equal(t, "redacted", view["tool_argument_error_mode"])
	masked, err := json.Marshal(view)
	require.NoError(t, err)
	require.NotContains(t, string(masked), "synthetic-private-token")
	require.Equal(t, "redacted", oauth.RedactServerConfigSecrets(storedAfter, oauth.LiveRedaction).ToolArgumentErrorMode)
}
