package runtime

import (
	"path/filepath"
	"testing"

	"github.com/smart-mcp-proxy/mcpproxy-go/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// RUNTIME-MODE-01: the real two-store lifecycle must keep the stored policy
// while rebuilding JSON, reloading disk edits, and saving an unrelated server.
// Disabled loopback fixtures never connect to an external MCP service.
func TestArgumentErrorModeSaveReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp_config.json")
	cfg := config.DefaultConfig()
	cfg.DataDir = dir
	cfg.Tokenizer = &config.TokenizerConfig{Enabled: false}
	cfg.Servers = []*config.ServerConfig{
		{Name: "private-upstream", URL: "http://127.0.0.1:1/mcp", Protocol: "streamable-http", Enabled: false},
		{Name: "control", URL: "http://127.0.0.1:1/control", Protocol: "streamable-http", Enabled: false, Headers: map[string]string{"Authorization": "synthetic-control"}},
	}
	require.NoError(t, config.SaveConfig(cfg, path))
	rt, err := New(cfg, path, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close()) })
	for _, sc := range cfg.Servers {
		stored := config.CopyServerConfig(sc)
		if stored.Name == "private-upstream" {
			stored.ToolArgumentErrorMode = "redacted"
		}
		require.NoError(t, rt.StorageManager().SaveUpstreamServer(stored))
	}
	verify := func(mode string) {
		disk, err := config.LoadFromFile(path)
		require.NoError(t, err)
		stored, err := rt.StorageManager().GetUpstreamServer("private-upstream")
		require.NoError(t, err)
		require.Equal(t, mode, stored.ToolArgumentErrorMode)
		for _, state := range []*config.Config{disk, rt.Config(), rt.ConfigSnapshot().Config} {
			require.Len(t, state.Servers, 2)
			found := false
			for _, sc := range state.Servers {
				if sc.Name == "private-upstream" {
					found = true
					require.Equal(t, mode, sc.ToolArgumentErrorMode)
				}
				if sc.Name == "control" {
					require.Empty(t, sc.ToolArgumentErrorMode)
					require.False(t, sc.Enabled)
					require.Equal(t, "synthetic-control", sc.Headers["Authorization"])
				}
			}
			require.True(t, found)
		}
	}
	require.NoError(t, rt.SaveConfiguration())
	verify("redacted")
	for _, mode := range []string{"detailed", "redacted"} {
		disk, err := config.LoadFromFile(path)
		require.NoError(t, err)
		for _, sc := range disk.Servers {
			if sc.Name == "private-upstream" {
				sc.ToolArgumentErrorMode = mode
			}
		}
		require.NoError(t, config.SaveConfig(disk, path))
		require.NoError(t, rt.ReloadConfiguration())
		verify(mode)
	}
	control, err := rt.StorageManager().GetUpstreamServer("control")
	require.NoError(t, err)
	control.URL = "http://127.0.0.1:1/updated-control"
	require.NoError(t, rt.StorageManager().SaveUpstreamServer(control))
	require.NoError(t, rt.SaveConfiguration())
	verify("redacted")
	unchanged, err := rt.StorageManager().GetUpstreamServer("control")
	require.NoError(t, err)
	require.Equal(t, control.URL, unchanged.URL)
	require.Equal(t, control.Headers, unchanged.Headers)
}
