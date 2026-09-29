package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"
)

// prompt_cache 三态必须原样穿过 JSON 往返：未配置=nil（沿用默认，由 litellm
// 的端点能力门控决定发不发）、false=显式关掉、true=显式声明端点接受。
func TestProviderConfigPromptCacheTriState(t *testing.T) {
	var cfg Config
	input := `{"providers":{
    "off":{"prompt_cache":false},
    "on":{"prompt_cache":true},
    "unset":{}
  }}`
	if err := json.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p := cfg.Providers["off"].PromptCache; p == nil || *p {
		t.Fatalf("off 应解码为 false, got %v", p)
	}
	if p := cfg.Providers["on"].PromptCache; p == nil || !*p {
		t.Fatalf("on 应解码为 true, got %v", p)
	}
	if p := cfg.Providers["unset"].PromptCache; p != nil {
		t.Fatalf("unset 应保持 nil, got %v", p)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), `"prompt_cache":null`) {
		t.Fatalf("nil 不该写回请求/配置: %s", data)
	}
	if !strings.Contains(string(data), `"prompt_cache":false`) {
		t.Fatalf("false 应原样写回: %s", data)
	}
}

// 只有显式 false 才算"关"。nil 与 true 都沿用默认（照常带上）——能力门控在
// 上游已经把端点判定纳入考量，应用层再拿 true 去反向覆盖只会绕开那道门。
func TestProviderPromptCacheOffOnlyExplicitFalse(t *testing.T) {
	tr, fa := true, false
	cfg := Config{Providers: map[string]ProviderConfig{
		"off":    {PromptCache: &fa},
		"on":     {PromptCache: &tr},
		"unset":  {},
		"absent": {},
	}}
	cases := map[string]bool{
		"off": true, "on": false, "unset": false, "absent": false, "": false,
	}
	for provider, want := range cases {
		if got := cfg.ProviderPromptCacheOff(provider); got != want {
			t.Errorf("ProviderPromptCacheOff(%q) = %v, want %v", provider, got, want)
		}
	}
}
