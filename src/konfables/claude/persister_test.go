package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eminert/konfi/pkg/parser"
)

func setupTiers(t *testing.T) (global, local, project string) {
	t.Helper()
	dir := t.TempDir()
	global = filepath.Join(dir, "global", "settings.json")
	local = filepath.Join(dir, "local", "settings.local.json")
	project = filepath.Join(dir, "project", ".claude", "settings.json")
	return global, local, project
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return m
}

func TestMergePrecedence(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"theme": "dark",
		"model": "opus",
	})
	writeJSON(t, local, map[string]any{
		"theme": "light",
	})
	writeJSON(t, project, map[string]any{
		"theme": "solarized",
	})

	tp := NewTieredPersister(global, local, project)
	data, err := tp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var merged map[string]any
	if err := json.Unmarshal(data, &merged); err != nil {
		t.Fatal(err)
	}

	// project > local > global
	if merged["theme"] != "solarized" {
		t.Errorf("theme = %v, want solarized", merged["theme"])
	}
	// inherited from global
	if merged["model"] != "opus" {
		t.Errorf("model = %v, want opus", merged["model"])
	}
}

func TestArrayAtomicReplace(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"permissions": map[string]any{
			"allow": []string{"read", "write"},
		},
	})
	writeJSON(t, project, map[string]any{
		"permissions": map[string]any{
			"allow": []string{"exec"},
		},
	})

	tp := NewTieredPersister(global, local, project)
	data, err := tp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var merged map[string]any
	if err := json.Unmarshal(data, &merged); err != nil {
		t.Fatal(err)
	}

	perms := merged["permissions"].(map[string]any)
	allow := perms["allow"].([]any)

	// project replaces, not extends
	if len(allow) != 1 || allow[0] != "exec" {
		t.Errorf("allow = %v, want [exec]", allow)
	}
}

func TestSaveRoutesToCorrectTier(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"theme": "dark",
	})
	writeJSON(t, local, map[string]any{
		"verbose": true,
	})

	tp := NewTieredPersister(global, local, project)
	ctx := context.Background()

	original, err := tp.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// change theme (owned by global) and verbose (owned by local)
	var blob map[string]any
	json.Unmarshal(original, &blob)
	blob["theme"] = "monokai"
	blob["verbose"] = false
	newData, _ := json.MarshalIndent(blob, "", "  ")
	newData = append(newData, '\n')

	if err := tp.Save(ctx, original, newData); err != nil {
		t.Fatal(err)
	}

	// check global file has updated theme
	gm := readJSON(t, global)
	if gm["theme"] != "monokai" {
		t.Errorf("global theme = %v, want monokai", gm["theme"])
	}

	// check local file has updated verbose
	lm := readJSON(t, local)
	if lm["verbose"] != false {
		t.Errorf("local verbose = %v, want false", lm["verbose"])
	}
}

func TestDeleteRemovesFromOwningTier(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"theme": "dark",
	})
	writeJSON(t, local, map[string]any{
		"theme": "light",
	})

	tp := NewTieredPersister(global, local, project)
	ctx := context.Background()

	original, err := tp.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// the merged blob has theme=light (from local)
	// delete theme from the merged blob
	var blob map[string]any
	json.Unmarshal(original, &blob)
	delete(blob, "theme")
	newData, _ := json.MarshalIndent(blob, "", "  ")
	newData = append(newData, '\n')

	if err := tp.Save(ctx, original, newData); err != nil {
		t.Fatal(err)
	}

	// local should no longer have theme (it was the owner)
	lm := readJSON(t, local)
	if _, ok := lm["theme"]; ok {
		t.Error("local still has theme after delete")
	}

	// global should still have theme
	gm := readJSON(t, global)
	if gm["theme"] != "dark" {
		t.Errorf("global theme = %v, want dark (inherited)", gm["theme"])
	}
}

func TestDeleteSurfacesInheritedValue(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"theme": "dark",
	})
	writeJSON(t, local, map[string]any{
		"theme": "light",
	})

	tp := NewTieredPersister(global, local, project)
	ctx := context.Background()

	original, err := tp.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// delete theme (owned by local)
	var blob map[string]any
	json.Unmarshal(original, &blob)
	delete(blob, "theme")
	newData, _ := json.MarshalIndent(blob, "", "  ")
	newData = append(newData, '\n')

	if err := tp.Save(ctx, original, newData); err != nil {
		t.Fatal(err)
	}

	// re-load: theme should surface from global
	reloaded, err := tp.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var merged map[string]any
	json.Unmarshal(reloaded, &merged)
	if merged["theme"] != "dark" {
		t.Errorf("after delete+reload, theme = %v, want dark", merged["theme"])
	}
}

func TestNewKeysDefaultToGlobal(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"theme": "dark",
	})

	tp := NewTieredPersister(global, local, project)
	ctx := context.Background()

	original, err := tp.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// add a brand-new key
	var blob map[string]any
	json.Unmarshal(original, &blob)
	blob["newkey"] = "newvalue"
	newData, _ := json.MarshalIndent(blob, "", "  ")
	newData = append(newData, '\n')

	if err := tp.Save(ctx, original, newData); err != nil {
		t.Fatal(err)
	}

	gm := readJSON(t, global)
	if gm["newkey"] != "newvalue" {
		t.Errorf("global newkey = %v, want newvalue", gm["newkey"])
	}
}

func TestTierMapProvenance(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"theme": "dark",
		"model": "opus",
	})
	writeJSON(t, local, map[string]any{
		"theme": "light",
	})
	writeJSON(t, project, map[string]any{
		"theme": "solarized",
	})

	tp := NewTieredPersister(global, local, project)
	if _, err := tp.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	// theme is defined in all three tiers, project wins
	if tier := tp.TierOf("theme"); tier != TierProject {
		t.Errorf("TierOf(theme) = %s, want %s", tier, TierProject)
	}

	// model only in global
	if tier := tp.TierOf("model"); tier != TierGlobal {
		t.Errorf("TierOf(model) = %s, want %s", tier, TierGlobal)
	}
}

func TestTiersReturnsAllDefiningTiers(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"theme": "dark",
	})
	writeJSON(t, local, map[string]any{
		"theme": "light",
	})
	writeJSON(t, project, map[string]any{
		"theme": "solarized",
	})

	tp := NewTieredPersister(global, local, project)
	if _, err := tp.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	tiers := tp.Tiers("theme")
	if len(tiers) != 3 {
		t.Fatalf("Tiers(theme) = %v, want 3 tiers", tiers)
	}
	// highest precedence first
	if tiers[0] != TierProject || tiers[1] != TierLocal || tiers[2] != TierGlobal {
		t.Errorf("Tiers(theme) = %v, want [project local global]", tiers)
	}
}

func TestMissingFilesHandledGracefully(t *testing.T) {
	global, local, project := setupTiers(t)

	// only write global, leave local and project missing
	writeJSON(t, global, map[string]any{
		"theme": "dark",
	})

	tp := NewTieredPersister(global, local, project)
	data, err := tp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var merged map[string]any
	if err := json.Unmarshal(data, &merged); err != nil {
		t.Fatal(err)
	}

	if merged["theme"] != "dark" {
		t.Errorf("theme = %v, want dark", merged["theme"])
	}
}

func TestAllFilesMissing(t *testing.T) {
	global, local, project := setupTiers(t)

	tp := NewTieredPersister(global, local, project)
	data, err := tp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var merged map[string]any
	if err := json.Unmarshal(data, &merged); err != nil {
		t.Fatal(err)
	}

	if len(merged) != 0 {
		t.Errorf("expected empty merged, got %v", merged)
	}
}

func TestDeepMergeNestedObjects(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"permissions": map[string]any{
			"allow": []string{"read"},
			"deny":  []string{"exec"},
		},
	})
	writeJSON(t, project, map[string]any{
		"permissions": map[string]any{
			"allow": []string{"write"},
		},
	})

	tp := NewTieredPersister(global, local, project)
	data, err := tp.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var merged map[string]any
	json.Unmarshal(data, &merged)

	perms := merged["permissions"].(map[string]any)

	// allow is replaced atomically by project
	allow := perms["allow"].([]any)
	if len(allow) != 1 || allow[0] != "write" {
		t.Errorf("allow = %v, want [write]", allow)
	}

	// deny is inherited from global (not touched by project)
	deny := perms["deny"].([]any)
	if len(deny) != 1 || deny[0] != "exec" {
		t.Errorf("deny = %v, want [exec]", deny)
	}
}

func TestTierOfUnknownKeyEmpty(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{"theme": "dark"})

	tp := NewTieredPersister(global, local, project)
	tp.Load(context.Background())

	if tier := tp.TierOf("nonexistent"); tier != "" {
		t.Errorf("TierOf(nonexistent) = %s, want empty", tier)
	}
}

func TestTiersOfUnknownKeyNil(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{"theme": "dark"})

	tp := NewTieredPersister(global, local, project)
	tp.Load(context.Background())

	if tiers := tp.Tiers("nonexistent"); tiers != nil {
		t.Errorf("Tiers(nonexistent) = %v, want nil", tiers)
	}
}

func TestSaveNestedKeyToCorrectTier(t *testing.T) {
	global, local, project := setupTiers(t)

	writeJSON(t, global, map[string]any{
		"permissions": map[string]any{
			"allow": []string{"read"},
		},
	})
	writeJSON(t, project, map[string]any{
		"permissions": map[string]any{
			"deny": []string{"exec"},
		},
	})

	tp := NewTieredPersister(global, local, project)
	ctx := context.Background()

	original, err := tp.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// change the global allow array
	var blob map[string]any
	json.Unmarshal(original, &blob)
	blob["permissions"].(map[string]any)["allow"] = []any{"read", "write"}
	newData, _ := json.MarshalIndent(blob, "", "  ")
	newData = append(newData, '\n')

	if err := tp.Save(ctx, original, newData); err != nil {
		t.Fatal(err)
	}

	// global should have updated allow
	gm := readJSON(t, global)
	gPerms := gm["permissions"].(map[string]any)
	gAllow := gPerms["allow"].([]any)
	if len(gAllow) != 2 {
		t.Errorf("global allow = %v, want [read write]", gAllow)
	}
}

// TestIntegration3Tier exercises the full lifecycle: 3-tier merge, edit via parser,
// save isolation, delete+resurface, and tier provenance tracking.
func TestIntegration3Tier(t *testing.T) {
	global, local, project := setupTiers(t)
	ctx := context.Background()
	parser := &parser.JSONParser{}

	// --- fixtures: 3 tiers with overlapping keys ---
	writeJSON(t, global, map[string]any{
		"effortLevel": "high",
		"permissions": map[string]any{
			"allow": []string{"Read"},
		},
	})
	writeJSON(t, local, map[string]any{
		"model": "opus",
	})
	writeJSON(t, project, map[string]any{
		"effortLevel": "max",
		"permissions": map[string]any{
			"allow": []string{"Bash(*)"},
		},
	})

	tp := NewTieredPersister(global, local, project)

	// --- step 1: verify merged load precedence ---
	data, err := tp.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	val, ok := parser.FindValue(data, "effortLevel")
	if !ok || val != "max" {
		t.Fatalf("merged effortLevel = %q, want max", val)
	}

	vals, ok := parser.FindValues(data, "permissions.allow")
	if !ok || len(vals) != 1 || vals[0] != "Bash(*)" {
		t.Fatalf("merged permissions.allow = %v, want [Bash(*)]", vals)
	}

	val, ok = parser.FindValue(data, "model")
	if !ok || val != "opus" {
		t.Fatalf("merged model = %q, want opus", val)
	}

	// tier provenance after load
	if tier := tp.TierOf("effortLevel"); tier != TierProject {
		t.Errorf("TierOf(effortLevel) = %s, want %s", tier, TierProject)
	}
	if tier := tp.TierOf("permissions.allow"); tier != TierProject {
		t.Errorf("TierOf(permissions.allow) = %s, want %s", tier, TierProject)
	}
	if tier := tp.TierOf("model"); tier != TierLocal {
		t.Errorf("TierOf(model) = %s, want %s", tier, TierLocal)
	}

	// --- step 2: edit permission via parser, add "Write" to project allow ---
	original := make([]byte, len(data))
	copy(original, data)

	data, err = parser.SetValues(data, "permissions.allow", []string{"Bash(*)", "Write"})
	if err != nil {
		t.Fatalf("SetValues: %v", err)
	}

	if err := tp.Save(ctx, original, data); err != nil {
		t.Fatalf("Save after SetValues: %v", err)
	}

	// only project file should have changed
	pm := readJSON(t, project)
	pPerms := pm["permissions"].(map[string]any)
	pAllow := pPerms["allow"].([]any)
	if len(pAllow) != 2 {
		t.Fatalf("project allow = %v, want [Bash(*) Write]", pAllow)
	}
	if pAllow[0] != "Bash(*)" || pAllow[1] != "Write" {
		t.Errorf("project allow = %v, want [Bash(*) Write]", pAllow)
	}

	// global should be unchanged
	gm := readJSON(t, global)
	gPerms := gm["permissions"].(map[string]any)
	gAllow := gPerms["allow"].([]any)
	if len(gAllow) != 1 || gAllow[0] != "Read" {
		t.Errorf("global allow = %v, want [Read] (unchanged)", gAllow)
	}

	// local should be unchanged
	lm := readJSON(t, local)
	if lm["model"] != "opus" {
		t.Errorf("local model = %v, want opus (unchanged)", lm["model"])
	}

	// --- step 3: delete project effortLevel, verify global value surfaces ---
	data, err = tp.Load(ctx)
	if err != nil {
		t.Fatalf("Load before delete: %v", err)
	}
	original = make([]byte, len(data))
	copy(original, data)

	data, err = parser.DeleteKey(data, "effortLevel")
	if err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}

	if err := tp.Save(ctx, original, data); err != nil {
		t.Fatalf("Save after delete: %v", err)
	}

	// project file should no longer have effortLevel
	pmRaw, err := os.ReadFile(project)
	if err != nil {
		t.Fatalf("read project: %v", err)
	}
	var pmAfter map[string]any
	json.Unmarshal(pmRaw, &pmAfter)
	if _, found := pmAfter["effortLevel"]; found {
		t.Error("project still has effortLevel after delete")
	}

	// global should still have effortLevel
	gmAfter := readJSON(t, global)
	if gmAfter["effortLevel"] != "high" {
		t.Errorf("global effortLevel = %v, want high", gmAfter["effortLevel"])
	}

	// re-load: global value should surface
	reloaded, err := tp.Load(ctx)
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
	}
	val, ok = parser.FindValue(reloaded, "effortLevel")
	if !ok || val != "high" {
		t.Fatalf("after delete+reload, effortLevel = %q, want high", val)
	}

	// tier provenance should now point to global
	if tier := tp.TierOf("effortLevel"); tier != TierGlobal {
		t.Errorf("TierOf(effortLevel) after delete = %s, want %s", tier, TierGlobal)
	}
}
