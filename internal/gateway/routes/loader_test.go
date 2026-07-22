package routes

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTemp 写入临时路由文件并返回路径。
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routes.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp routes: %v", err)
	}
	return path
}

func TestLoadFile_valid(t *testing.T) {
	// 加载后需还原内置表，避免影响同包其他用例。
	prev := current.Load()
	defer current.Store(prev)

	path := writeTemp(t, `{
		"routes": [
			{"public_path": "/v1/foo", "upstream_path": "/foo", "service_name": "foo-service", "requires_auth": false},
			{"public_path": "/v1/bar/", "upstream_path": "/bar/", "service_name": "bar-service", "requires_auth": true, "match": "prefix"},
			{"public_path": "/v1/direct", "upstream_path": "/d", "upstream_base_url": "http://127.0.0.1:9000", "requires_auth": false}
		]
	}`)

	if err := Reload(path); err != nil {
		t.Fatalf("Reload valid file: %v", err)
	}

	// 精确匹配。
	if r, ok := Match("/v1/foo"); !ok || r.UpstreamPath != "/foo" || r.Prefix {
		t.Fatalf("exact match failed: %+v ok=%v", r, ok)
	}
	// 前缀匹配 + 尾段。
	r, ok := Match("/v1/bar/123")
	if !ok || !r.Prefix || !r.RequiresAuth {
		t.Fatalf("prefix match failed: %+v ok=%v", r, ok)
	}
	// 直连基址路由。
	if r, ok := Match("/v1/direct"); !ok || r.UpstreamBaseURL != "http://127.0.0.1:9000" {
		t.Fatalf("direct base match failed: %+v ok=%v", r, ok)
	}
	// 未命中。
	if _, ok := Match("/v1/nope"); ok {
		t.Fatalf("unexpected match for /v1/nope")
	}
}

func TestLoadFile_invalidKeepsPrevious(t *testing.T) {
	prev := current.Load()
	defer current.Store(prev)

	cases := []string{
		`{"routes": []}`,
		`{"routes": [{"public_path": "v1/foo", "upstream_path": "/foo", "service_name": "s"}]}`,                     // public_path 不以 / 开头
		`{"routes": [{"public_path": "/v1/foo", "upstream_path": "/foo"}]}`,                                         // 缺 service_name 与 base
		`{"routes": [{"public_path": "/v1/foo", "upstream_path": "/foo", "service_name": "s", "match": "prefix"}]}`, // 前缀 public_path 不以 / 结尾
		`{"routes": [{"public_path": "/v1/foo", "upstream_path": "/foo", "service_name": "s", "match": "weird"}]}`,  // 非法 match
		`not-json`,
	}
	for i, c := range cases {
		path := writeTemp(t, c)
		if err := Reload(path); err == nil {
			t.Fatalf("case %d: expected error, got nil", i)
		}
	}
	// 校验失败后当前表必须仍是加载前的旧表（坏配置不打挂网关）。
	if current.Load() != prev {
		t.Fatalf("invalid reload must keep previous table")
	}
}

func TestReload_missingFile(t *testing.T) {
	if err := Reload(filepath.Join(t.TempDir(), "does-not-exist.json")); err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestBuiltinDefaults(t *testing.T) {
	// 确保内置默认表可用（开箱即用）。
	current.Store(builtinTable())
	if !RequiresAuth("/v1/order/orders/123") {
		t.Fatalf("builtin: /v1/order/orders/123 should require auth")
	}
	if RequiresAuth("/v1/auth/login") {
		t.Fatalf("builtin: /v1/auth/login should be public")
	}
	if Count() == 0 {
		t.Fatalf("builtin: count should be > 0")
	}
}

// TestHotReload_liveRouting 模拟运行时热加载：改文件 → Reload → Match 立即反映新路由，
// 且下游目标路径按精确/前缀正确拼接（等价于网关 dispatch 的路由决策）。
func TestHotReload_liveRouting(t *testing.T) {
	prev := current.Load()
	defer current.Store(prev)

	// 第一版：仅 /v1/a 精确路由。
	pathV1 := writeTemp(t, `{"routes":[
		{"public_path":"/v1/a","upstream_path":"/a","service_name":"svc-a"}
	]}`)
	if err := Reload(pathV1); err != nil {
		t.Fatalf("reload v1: %v", err)
	}
	if _, ok := Match("/v1/b/42"); ok {
		t.Fatalf("v1 should not match /v1/b/42 yet")
	}
	ra, _ := Match("/v1/a")
	if got := ra.UpstreamTargetPath("/v1/a"); got != "/a" {
		t.Fatalf("exact target: want /a got %s", got)
	}

	// 第二版：热加载后新增 /v1/b/ 前缀路由。
	pathV2 := writeTemp(t, `{"routes":[
		{"public_path":"/v1/a","upstream_path":"/a","service_name":"svc-a"},
		{"public_path":"/v1/b/","upstream_path":"/b/","service_name":"svc-b","match":"prefix"}
	]}`)
	if err := Reload(pathV2); err != nil {
		t.Fatalf("reload v2: %v", err)
	}
	rb, ok := Match("/v1/b/42")
	if !ok {
		t.Fatalf("v2 should match /v1/b/42 after hot reload")
	}
	if got := rb.UpstreamTargetPath("/v1/b/42"); got != "/b/42" {
		t.Fatalf("prefix target: want /b/42 got %s", got)
	}
}
