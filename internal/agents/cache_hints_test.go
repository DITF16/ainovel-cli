package agents

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
)

// 中转把 cache_control 原样透传给上游、上游模型对断点直接 400 时，
// "prompt_cache": false 必须让该 provider 的两个字段同时归零——只清一个仍会
// 带着另一半出门（断点被拒与 prompt_cache_key 被拒是两类错）。反过来，未配置
// 与显式 true 都必须照常带上：这个开关是逃生口，不是默认值。
func TestResolveCacheHintsHonorsPerProviderPromptCache(t *testing.T) {
	off, on := false, true
	cfg := bootstrap.Config{
		Provider: "direct", ModelName: "m-direct",
		Providers: map[string]bootstrap.ProviderConfig{
			"direct": {Type: "openai", APIKey: "k", BaseURL: "https://api.example.com/v1", PromptCache: &on},
			"relay":  {Type: "openai", APIKey: "k", BaseURL: "https://relay.example.com/v1", PromptCache: &off},
			"plain":  {Type: "openai", APIKey: "k", BaseURL: "https://plain.example.com/v1"},
		},
		Roles: map[string]bootstrap.RoleConfig{
			"writer": {Provider: "relay", Model: "m-relay"},
			"editor": {Provider: "plain", Model: "m-plain"},
		},
	}
	models, err := bootstrap.NewModelSet(cfg)
	if err != nil {
		t.Fatalf("new model set: %v", err)
	}

	// writer 落在显式关掉的 relay 上：两个字段都得空。
	if h := resolveCacheHints(cfg, models, "writer", "nvl-x-writer"); h.lastMessage != "" || h.key != "" {
		t.Errorf("writer(prompt_cache:false) 应清空两个字段, got %+v", h)
	}
	// editor 未配置该字段、architect 落在显式 true 上：都照常带。
	if h := resolveCacheHints(cfg, models, "editor", "nvl-x-editor"); h.lastMessage != "ephemeral" || h.key != "nvl-x-editor" {
		t.Errorf("editor(未配置) 应照常带, got %+v", h)
	}
	if h := resolveCacheHints(cfg, models, "architect", "nvl-x-architect"); h.lastMessage != "ephemeral" || h.key != "nvl-x-architect" {
		t.Errorf("architect(prompt_cache:true) 应照常带, got %+v", h)
	}
	// models 缺失时不得崩，退回默认带上。
	if h := resolveCacheHints(cfg, nil, "writer", "nvl-x-writer"); h.lastMessage != "ephemeral" || h.key != "nvl-x-writer" {
		t.Errorf("models=nil 应退回默认, got %+v", h)
	}
}

// 缓存身份键是缓存血统：architect 的两个子代理必须仍是两个独立键，
// 不能因为共用 role 解析而被合并成同一个——那会让两个会话共用一个路由桶。
func TestResolveCacheHintsKeepsPerSubagentIdentity(t *testing.T) {
	cfg := bootstrap.Config{Provider: "direct", ModelName: "m"}
	short := resolveCacheHints(cfg, nil, "architect", "nvl-x-architect_short")
	long := resolveCacheHints(cfg, nil, "architect", "nvl-x-architect_long")
	if short.key == long.key {
		t.Fatalf("architect 两个子代理的缓存键不应相同: %q", short.key)
	}
	if short.key != "nvl-x-architect_short" || long.key != "nvl-x-architect_long" {
		t.Fatalf("缓存键应原样透传, got %q / %q", short.key, long.key)
	}
}
