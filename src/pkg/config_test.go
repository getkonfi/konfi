package pkg

import (
	"context"
	"fmt"
	"testing"
)

// mockPersister is a simple in-memory persister for testing ConfigFile.
type mockPersister struct {
	data    []byte
	saved   bool
	saveErr error
}

func (m *mockPersister) Load(_ context.Context) ([]byte, error) {
	return m.data, nil
}

func (m *mockPersister) Save(_ context.Context, _, data []byte) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.data = data
	m.saved = true
	return nil
}

func TestNewConfigFile(t *testing.T) {
	mp := &mockPersister{data: []byte("key = value\n")}
	cf, err := NewConfigFile(context.Background(), mp)
	if err != nil {
		t.Fatal(err)
	}
	if string(cf.Content()) != "key = value\n" {
		t.Errorf("unexpected content: %q", cf.Content())
	}
	if cf.Dirty() {
		t.Error("new config should not be dirty")
	}
}

func TestConfigFileDirtyTracking(t *testing.T) {
	mp := &mockPersister{data: []byte("original\n")}
	cf, _ := NewConfigFile(context.Background(), mp)

	cf.SetContent([]byte("modified\n"))
	if !cf.Dirty() {
		t.Error("expected dirty after SetContent")
	}

	cf.SetContent([]byte("original\n"))
	if cf.Dirty() {
		t.Error("expected clean after restoring original content")
	}
}

func TestConfigFileSave(t *testing.T) {
	mp := &mockPersister{data: []byte("original\n")}
	cf, _ := NewConfigFile(context.Background(), mp)

	cf.SetContent([]byte("modified\n"))
	if err := cf.Save(context.Background()); err != nil {
		t.Fatal(err)
	}

	if cf.Dirty() {
		t.Error("expected clean after save")
	}
	if !mp.saved {
		t.Error("expected persister.Save to be called")
	}
	if string(mp.data) != "modified\n" {
		t.Errorf("persister data = %q, want %q", mp.data, "modified\n")
	}
}

func TestConfigFileSaveError(t *testing.T) {
	mp := &mockPersister{data: []byte("original\n"), saveErr: fmt.Errorf("disk full")}
	cf, _ := NewConfigFile(context.Background(), mp)

	cf.SetContent([]byte("modified\n"))
	err := cf.Save(context.Background())
	if err == nil {
		t.Fatal("expected error from save")
	}
	if !cf.Dirty() {
		t.Error("expected still dirty after failed save")
	}
}

func TestConfigFileReload(t *testing.T) {
	mp := &mockPersister{data: []byte("v1\n")}
	cf, _ := NewConfigFile(context.Background(), mp)

	cf.SetContent([]byte("local edit\n"))
	if !cf.Dirty() {
		t.Error("expected dirty after local edit")
	}

	// simulate external change
	mp.data = []byte("v2\n")
	if err := cf.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}

	if cf.Dirty() {
		t.Error("expected clean after reload")
	}
	if string(cf.Content()) != "v2\n" {
		t.Errorf("expected reloaded content, got %q", cf.Content())
	}
}

type mockTierPersister struct {
	data  []byte
	tiers map[string]string
}

func (m *mockTierPersister) Load(_ context.Context) ([]byte, error) { return m.data, nil }
func (m *mockTierPersister) Save(_ context.Context, _, data []byte) error {
	m.data = data
	return nil
}
func (m *mockTierPersister) TierOf(key string) string { return m.tiers[key] }
func (m *mockTierPersister) Tiers(key string) []string {
	tier := m.tiers[key]
	if tier == "" {
		return nil
	}
	return []string{tier}
}

func TestConfigFileGeneration(t *testing.T) {
	mp := &mockPersister{data: []byte("v1\n")}
	cf, _ := NewConfigFile(context.Background(), mp)

	gen := cf.Generation()
	cf.SetContent([]byte("v2\n"))
	if cf.Generation() <= gen {
		t.Errorf("expected generation to increase, got %d then %d", gen, cf.Generation())
	}
}

func TestConfigFilePreview(t *testing.T) {
	mp := &mockPersister{data: []byte("original\n")}
	cf, _ := NewConfigFile(context.Background(), mp)

	cf.SetContent([]byte("previewed\n"))
	if err := cf.Preview(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !cf.Dirty() {
		t.Error("expected dirty after preview (original not promoted)")
	}
	if string(mp.data) != "previewed\n" {
		t.Errorf("persister data = %q, want previewed content", mp.data)
	}
}

func TestConfigFileRevertPreview(t *testing.T) {
	mp := &mockPersister{data: []byte("original\n")}
	cf, _ := NewConfigFile(context.Background(), mp)

	cf.SetContent([]byte("modified\n"))
	if err := cf.Preview(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := cf.RevertPreview(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !cf.Dirty() {
		t.Error("expected still dirty after revert preview")
	}
	if string(mp.data) != "original\n" {
		t.Errorf("persister data = %q, want original after revert", mp.data)
	}
}

func TestConfigFileTierOf(t *testing.T) {
	mp := &mockTierPersister{
		data:  []byte("{}"),
		tiers: map[string]string{"theme": "global"},
	}
	cf, _ := NewConfigFile(context.Background(), mp)

	got := cf.TierOf("theme")
	if got != "global" {
		t.Errorf("TierOf(theme) = %q, want global", got)
	}
	got = cf.TierOf("missing")
	if got != "" {
		t.Errorf("TierOf(missing) = %q, want empty", got)
	}
}

func TestConfigFileTiers(t *testing.T) {
	mp := &mockTierPersister{
		data:  []byte("{}"),
		tiers: map[string]string{"theme": "global"},
	}
	cf, _ := NewConfigFile(context.Background(), mp)

	got := cf.Tiers("theme")
	if len(got) != 1 || got[0] != "global" {
		t.Errorf("Tiers(theme) = %v, want [global]", got)
	}
	got = cf.Tiers("missing")
	if got != nil {
		t.Errorf("Tiers(missing) = %v, want nil", got)
	}
}

func TestConfigFileTierOfNonTierPersister(t *testing.T) {
	mp := &mockPersister{data: []byte("key = value\n")}
	cf, _ := NewConfigFile(context.Background(), mp)

	got := cf.TierOf("key")
	if got != "" {
		t.Errorf("TierOf on non-tier persister = %q, want empty", got)
	}
}

func TestConfigFileContentIsolation(t *testing.T) {
	mp := &mockPersister{data: []byte("original\n")}
	cf, _ := NewConfigFile(context.Background(), mp)

	content := cf.Content()
	content[0] = 'X'
	if string(cf.Content()) == "Xriginal\n" {
		t.Error("Content() should return a copy, not a reference")
	}
}
