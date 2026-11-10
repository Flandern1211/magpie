package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/provider"
)

// outputHome is syncHome with a model models.dev says replies 384000
// tokens, as it says of deepseek-v4.1-flash, served by the relay.
func outputHome(t *testing.T) string {
	t.Helper()
	home := syncHome(t)
	listOutput(t, 384000)
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"deepseek-v4.1-flash"}}); err != nil {
		t.Fatal(err)
	}
	return home
}

// listOutput is models.dev saying the model replies n tokens.
func listOutput(t *testing.T, n int) {
	t.Helper()
	os.WriteFile(catalog.CachePath(), []byte(`{"deepseek":{"models":{"deepseek-v4.1-flash":{"id":"deepseek-v4.1-flash","name":"DeepSeek V4.1 Flash","limit":{"context":1000000,"output":`+strconv.Itoa(n)+`}}}}}`), 0o644)
	catalog.Reset()
}

// sayagoodbyeplz on Discord: WorkBuddy's deepseek-v4.1-flash replies
// 128000 tokens, not the 384000 models.dev says, and a maxTokens set to
// 128000 by hand in Pi's models.json was written back to 384000 by the
// next sync. One typed there stays; a Max output set in magpie is written
// over it; and with neither, magpie's own follows models.dev.
func TestPiKeepsAMaxTokensTypedByHand(t *testing.T) {
	home := outputHome(t)
	piModels := filepath.Join(home, ".pi", "agent", "models.json")
	writeFile(t, filepath.Join(home, ".pi", "agent", "settings.json"), `{"defaultProvider":"magpie","defaultModel":"relay/deepseek-v4.1-flash"}`)
	writeFile(t, piModels, `{"providers":{"magpie":{"name":"magpie","models":[]}}}`)
	setUpInMagpie("pi")
	const at = "providers.magpie.models.0.maxTokens"
	got := func() string { v, _ := edit.GetJSON(piModels, at); return v }
	SyncCatalog()
	if got() != "384000" {
		t.Fatalf("magpie's maxTokens = %s, want models.dev's 384000\n%s", got(), readFile(piModels))
	}

	// models.dev changes, nothing typed: magpie's follows it
	listOutput(t, 300000)
	SyncCatalog()
	if got() != "300000" {
		t.Fatalf("after models.dev changed, maxTokens = %s, want 300000", got())
	}

	// typed by hand: it stays through syncs, and models.dev changing
	if err := edit.SetJSON(piModels, edit.KV{Path: at, Value: 128000}); err != nil {
		t.Fatal(err)
	}
	SyncCatalog()
	if got() != "128000" {
		t.Fatalf("a maxTokens typed by hand was written over: %s", got())
	}
	listOutput(t, 384000)
	SyncCatalog()
	if got() != "128000" {
		t.Fatalf("a maxTokens typed by hand was written over when models.dev changed: %s", got())
	}

	// set in magpie: the user's word there is written over it
	if err := provider.SetModelOutput("relay/deepseek-v4.1-flash", 100000); err != nil {
		t.Fatal(err)
	}
	SyncCatalog()
	if got() != "100000" {
		t.Fatalf("a Max output set in magpie didn't reach Pi: %s", got())
	}
	// and taken away, magpie's own again
	if err := provider.SetModelOutput("relay/deepseek-v4.1-flash", 0); err != nil {
		t.Fatal(err)
	}
	SyncCatalog()
	if got() != "384000" {
		t.Fatalf("with no Max output set, maxTokens = %s, want 384000", got())
	}
}

// The same for OpenCode's limit.output.
func TestOpenCodeKeepsALimitOutputTypedByHand(t *testing.T) {
	home := outputHome(t)
	cfg := filepath.Join(home, ".config", "opencode", "opencode.json")
	writeFile(t, cfg, `{"model":"magpie/relay/deepseek-v4.1-flash","provider":{"magpie":{"name":"magpie","npm":"@ai-sdk/openai-compatible","options":{"baseURL":"http://127.0.0.1:3425/v1","apiKey":"magpie"},"models":{}}}}`)
	oc := opencode(home, filepath.Join(home, ".config"))
	const at = "provider.magpie.models.relay/deepseek-v4\\.1-flash.limit.output"
	got := func() string { v, _ := edit.GetJSON(cfg, at); return v }
	if err := oc.Sync(); err != nil {
		t.Fatal(err)
	}
	if got() != "384000" {
		t.Fatalf("magpie's limit.output = %q, want 384000\n%s", got(), readFile(cfg))
	}
	if err := edit.SetJSON(cfg, edit.KV{Path: at, Value: 128000}); err != nil {
		t.Fatal(err)
	}
	if err := oc.Sync(); err != nil {
		t.Fatal(err)
	}
	if got() != "128000" {
		t.Fatalf("a limit.output typed by hand was written over: %s", got())
	}
	if err := provider.SetModelOutput("relay/deepseek-v4.1-flash", 100000); err != nil {
		t.Fatal(err)
	}
	if err := oc.Sync(); err != nil {
		t.Fatal(err)
	}
	if got() != "100000" {
		t.Fatalf("a Max output set in magpie didn't reach OpenCode: %s", got())
	}
}

// The same for ZCode's two files: config.json's limit.output and
// provider_config.json's optionSpecs.maxOutputTokens.max.
func TestZCodeKeepsALimitOutputTypedByHand(t *testing.T) {
	home := outputHome(t)
	zc := zcode(home)
	if err := zc.Field("provider").Set("magpie"); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".zcode", "v2", "config.json")
	rules := filepath.Join(home, ".zcode", "v2", "provider_config.json")
	const at = `provider.magpie.models.relay/deepseek-v4\.1-flash.limit.output`
	got := func() string { v, _ := edit.GetJSON(cfg, at); return v }
	if got() != "128000" {
		t.Fatalf("magpie's limit.output = %q, want 128000\n%s", got(), readFile(cfg))
	}
	if err := edit.SetJSON(cfg, edit.KV{Path: at, Value: 64000}); err != nil {
		t.Fatal(err)
	}
	setZcodeRuleOutput(t, rules, "relay/deepseek-v4.1-flash", 64000)
	if err := zc.Sync(); err != nil {
		t.Fatal(err)
	}
	if got() != "64000" {
		t.Fatalf("a limit.output typed by hand was written over: %s", got())
	}
	if n := zcodeRuleOutputOf(t, rules, "relay/deepseek-v4.1-flash"); n != 64000 {
		t.Fatalf("a maxOutputTokens typed by hand was written over: %d", n)
	}
	if err := provider.SetModelOutput("relay/deepseek-v4.1-flash", 100000); err != nil {
		t.Fatal(err)
	}
	if err := zc.Sync(); err != nil {
		t.Fatal(err)
	}
	if got() != "100000" {
		t.Fatalf("a Max output set in magpie didn't reach ZCode: %s", got())
	}
	if n := zcodeRuleOutputOf(t, rules, "relay/deepseek-v4.1-flash"); n != 100000 {
		t.Fatalf("a Max output set in magpie didn't reach ZCode's rules: %d", n)
	}
}

// dsh's maxTokens in the route of the profile it writes.
func TestDshKeepsAMaxTokensTypedByHand(t *testing.T) {
	home, _, web := dshRouteHome(t)
	os.MkdirAll(filepath.Dir(catalog.CachePath()), 0o755)
	listOutput(t, 384000)
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"deepseek-v4.1-flash"}}); err != nil {
		t.Fatal(err)
	}
	a := dsh(home)
	const ref = "relay/deepseek-v4.1-flash"
	if err := a.Field("model").Set("magpie/" + ref); err != nil {
		t.Fatal(err)
	}
	got := func() int { return dshRouteMaxTokens(t, web, ref) }
	if got() != 384000 {
		t.Fatalf("magpie's maxTokens = %d, want 384000\n%s", got(), readFile(web))
	}
	b, _ := os.ReadFile(web)
	os.WriteFile(web, []byte(strings.Replace(string(b), "maxTokens: 384000", "maxTokens: 128000", 1)), 0o644)
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	if got() != 128000 {
		t.Fatalf("a maxTokens typed by hand was written over: %d\n%s", got(), readFile(web))
	}
	if err := provider.SetModelOutput(ref, 100000); err != nil {
		t.Fatal(err)
	}
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	if got() != 100000 {
		t.Fatalf("a Max output set in magpie didn't reach dsh: %d", got())
	}
	if err := provider.SetModelOutput(ref, 0); err != nil {
		t.Fatal(err)
	}
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	if got() != 384000 {
		t.Fatalf("with no Max output set, maxTokens = %d, want 384000", got())
	}
}

// dshRouteMaxTokens is the maxTokens of id on magpie's route in the patch
// list at path.
func dshRouteMaxTokens(t *testing.T, path, id string) int {
	t.Helper()
	b, _ := os.ReadFile(path)
	var list []struct {
		ID     string `yaml:"id"`
		Config struct {
			Providers map[string]struct {
				Models []struct {
					ID        string `yaml:"id"`
					MaxTokens int    `yaml:"maxTokens"`
				} `yaml:"models"`
			} `yaml:"providers"`
		} `yaml:"config"`
	}
	if err := yaml.Unmarshal(b, &list); err != nil {
		t.Fatalf("%v:\n%s", err, b)
	}
	for _, it := range list {
		if it.ID != dshPiRow {
			continue
		}
		for _, m := range it.Config.Providers[dshRoute].Models {
			if m.ID == id {
				return m.MaxTokens
			}
		}
	}
	return 0
}

// zcodeRuleOutputOf is the maxOutputTokens magpie's rule for id has in
// provider_config.json at path.
func zcodeRuleOutputOf(t *testing.T, path, id string) int {
	t.Helper()
	b, _ := os.ReadFile(path)
	var doc struct {
		Config struct {
			ModelConfigRules struct {
				ProviderModelRules []struct {
					ModelID string `json:"modelId"`
					Config  struct {
						OptionSpecs struct {
							MaxOutputTokens struct {
								Max int `json:"max"`
							} `json:"maxOutputTokens"`
						} `json:"optionSpecs"`
					} `json:"config"`
				} `json:"providerModelRules"`
			} `json:"modelConfigRules"`
		} `json:"config"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%v:\n%s", err, b)
	}
	for _, r := range doc.Config.ModelConfigRules.ProviderModelRules {
		if r.ModelID == id {
			return r.Config.OptionSpecs.MaxOutputTokens.Max
		}
	}
	return 0
}

// setZcodeRuleOutput writes maxOutputTokens n into magpie's rule for id in
// provider_config.json at path.
func setZcodeRuleOutput(t *testing.T, path, id string, n int) {
	t.Helper()
	b, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%v:\n%s", err, b)
	}
	cfg, _ := doc["config"].(map[string]any)
	mcr, _ := cfg["modelConfigRules"].(map[string]any)
	rules, _ := mcr["providerModelRules"].([]any)
	for _, r := range rules {
		m, _ := r.(map[string]any)
		if m["modelId"] != id {
			continue
		}
		rc, _ := m["config"].(map[string]any)
		specs, _ := rc["optionSpecs"].(map[string]any)
		mt, _ := specs["maxOutputTokens"].(map[string]any)
		mt["max"] = n
	}
	out, _ := json.Marshal(doc)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
}
