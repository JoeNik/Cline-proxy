package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const officialModelsURL = "https://api.cline.bot/api/v1/ai/cline/recommended-models"

// ModelEntry is one model exposed by the proxy. Category mirrors the official
// model list groups; cost maps to the admin panel's free/pass/paid badges.
type ModelEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Category    string   `json:"category"`
	Cost        string   `json:"cost"`
	Status      string   `json:"status"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	OwnedBy     string   `json:"ownedBy"`
}

type modelStore struct {
	Models []ModelEntry `json:"models"`
}

type officialModel struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type officialModelsResponse struct {
	Recommended []officialModel `json:"recommended"`
	Free        []officialModel `json:"free"`
	Pass        []officialModel `json:"clinePass"`
	Cloud       []officialModel `json:"clineCloud"`
}

var (
	modelsData *modelStore
	modelsMu   sync.Mutex
	modelsPath string
)

func init() {
	exe, _ := os.Executable()
	modelsPath = filepath.Join(filepath.Dir(exe), ".cline-models.json")
}

func defaultModelEntries() []ModelEntry {
	return []ModelEntry{
		{ID: "cline-free/deepseek-v4.1-flash", Name: "Deepseek-v4.1-Flash", Provider: "deepseek", Category: "free", Cost: "free", Status: "active", OwnedBy: "cline"},
		{ID: "cline-free/muse-spark-1.3-contributor", Name: "Muse Spark 1.3 Contributor", Provider: "meta", Category: "free", Cost: "free", Status: "active", OwnedBy: "cline"},
		{ID: "cline-free/solar-pro4", Name: "Solar Pro 4", Provider: "upstage", Category: "free", Cost: "free", Status: "active", OwnedBy: "cline"},
		{ID: "z-ai/glm-5.3-flash", Name: "glm-5.3-flash", Provider: "zai", Category: "free", Cost: "free", Status: "active", OwnedBy: "zai"},
		{ID: "poolside/laguna-s-2.1:free", Name: "laguna-s-2.1:free", Provider: "poolside", Category: "free", Cost: "free", Status: "active", OwnedBy: "poolside"},
		{ID: "cline-free/glm-5.2", Name: "glm-5.2", Provider: "zai", Category: "free", Cost: "free", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/glm-5.2", Name: "glm-5.2", Provider: "zai", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/deepseek-v4-flash", Name: "deepseek-v4-flash", Provider: "deepseek", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/deepseek-v4.1-flash", Name: "deepseek-v4.1-flash", Provider: "deepseek", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/deepseek-v4-pro", Name: "deepseek-v4-pro", Provider: "deepseek", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/glm-5.3", Name: "glm-5.3", Provider: "zai", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/glm-5.3-flash", Name: "glm-5.3-flash", Provider: "zai", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/kimi-k2.6", Name: "kimi-k2.6", Provider: "moonshot", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/kimi-k2.7-code", Name: "kimi-k2.7-code", Provider: "moonshot", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/kimi-k3", Name: "kimi-k3", Provider: "moonshot", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/mimo-v2.5", Name: "mimo-v2.5", Provider: "mimo", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/mimo-v2.5-pro", Name: "mimo-v2.5-pro", Provider: "mimo", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/minimax-m3", Name: "minimax-m3", Provider: "minimax", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/qwen3.7-max", Name: "qwen3.7-max", Provider: "qwen", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/qwen3.7-plus", Name: "qwen3.7-plus", Provider: "qwen", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-pass/qwen3.8-max", Name: "qwen3.8-max", Provider: "qwen", Category: "pass", Cost: "pass", Status: "active", OwnedBy: "cline"},
		{ID: "cline-cloud/kimi-k3", Name: "kimi-k3", Provider: "moonshot", Category: "clineCloud", Cost: "paid", Status: "active", OwnedBy: "cline"},
		{ID: "cline-cloud/deepseek-v4-flash", Name: "deepseek-v4-flash", Provider: "deepseek", Category: "clineCloud", Cost: "paid", Status: "active", OwnedBy: "cline"},
		{ID: "cline-cloud/glm-5.2", Name: "glm-5.2", Provider: "zai", Category: "clineCloud", Cost: "paid", Status: "active", OwnedBy: "cline"},
		{ID: "deepseek/deepseek-v4-flash", Name: "deepseek-v4-flash", Provider: "deepseek", Category: "direct", Cost: "paid", Status: "active", OwnedBy: "deepseek"},
	}
}

func loadModelsStore() *modelStore {
	modelsMu.Lock()
	defer modelsMu.Unlock()

	if modelsData != nil {
		return modelsData
	}

	store := &modelStore{Models: defaultModelEntries()}
	if data, err := os.ReadFile(modelsPath); err == nil {
		var loaded modelStore
		if json.Unmarshal(data, &loaded) == nil && len(loaded.Models) > 0 {
			store = &loaded
		}
	}
	if store.Models == nil {
		store.Models = []ModelEntry{}
	}
	modelsData = store
	return store
}

func persistModelsStoreLocked() {
	if modelsData == nil {
		return
	}
	data, err := json.MarshalIndent(modelsData, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal models: %v", err)
		return
	}
	if err := os.WriteFile(modelsPath, data, 0600); err != nil {
		log.Printf("Failed to save models: %v", err)
	}
}

func saveModelsStore() {
	modelsMu.Lock()
	defer modelsMu.Unlock()
	persistModelsStoreLocked()
}

func supportedModels() []ModelEntry {
	store := loadModelsStore()
	modelsMu.Lock()
	defer modelsMu.Unlock()
	out := make([]ModelEntry, len(store.Models))
	copy(out, store.Models)
	return out
}

func addSupportedModel(entry ModelEntry) error {
	store := loadModelsStore()
	modelsMu.Lock()
	defer modelsMu.Unlock()
	for _, m := range store.Models {
		if m.ID == entry.ID {
			return fmt.Errorf("model already supported")
		}
	}
	store.Models = append(store.Models, entry)
	persistModelsStoreLocked()
	return nil
}

func removeSupportedModel(id string) error {
	store := loadModelsStore()
	modelsMu.Lock()
	defer modelsMu.Unlock()
	for i, m := range store.Models {
		if m.ID == id {
			store.Models = append(store.Models[:i], store.Models[i+1:]...)
			persistModelsStoreLocked()
			return nil
		}
	}
	return fmt.Errorf("model not found")
}

// removeSupportedModels removes every model whose id is in ids and returns the
// number of entries actually removed. Unknown ids are ignored.
func removeSupportedModels(ids []string) int {
	store := loadModelsStore()
	modelsMu.Lock()
	defer modelsMu.Unlock()

	removeSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		removeSet[id] = true
	}
	kept := make([]ModelEntry, 0, len(store.Models))
	removed := 0
	for _, m := range store.Models {
		if removeSet[m.ID] {
			removed++
			continue
		}
		kept = append(kept, m)
	}
	if removed > 0 {
		store.Models = kept
		persistModelsStoreLocked()
	}
	return removed
}

// clearSupportedModels empties the supported model list and returns the
// number of entries that were removed.
func clearSupportedModels() int {
	store := loadModelsStore()
	modelsMu.Lock()
	defer modelsMu.Unlock()

	removed := len(store.Models)
	if removed == 0 {
		return 0
	}
	store.Models = []ModelEntry{}
	persistModelsStoreLocked()
	return removed
}

// modelCostForCategory mirrors the official list groups to the proxy cost tiers.
func modelCostForCategory(category string) string {
	switch category {
	case "free":
		return "free"
	case "clinePass":
		return "pass"
	default:
		return "paid"
	}
}

func supportedModelsOpenAI() []map[string]any {
	models := supportedModels()
	out := make([]map[string]any, 0, len(models))
	created := time.Now().UnixMilli()
	for _, m := range models {
		out = append(out, map[string]any{
			"id":       m.ID,
			"object":   "model",
			"created":  created,
			"owned_by": m.OwnedBy,
		})
	}
	return out
}

func officialModelGroup(category, label string, items []officialModel, supported []ModelEntry) map[string]any {
	supportedSet := make(map[string]bool, len(supported))
	for _, m := range supported {
		supportedSet[m.ID] = true
	}
	models := make([]map[string]any, 0, len(items))
	for _, item := range items {
		models = append(models, map[string]any{
			"id":          item.ID,
			"name":        item.Name,
			"description": item.Description,
			"tags":        item.Tags,
			"category":    category,
			"provider":    providerFromID(item.ID),
			"supported":   supportedSet[item.ID],
		})
	}
	return map[string]any{"category": category, "label": label, "models": models}
}

// providerFromID extracts the provider segment before the first "/" of a model id.
func providerFromID(id string) string {
	if i := strings.Index(id, "/"); i > 0 {
		return id[:i]
	}
	return "unknown"
}

// fetchOfficialModels calls the official Cline recommended-models endpoint and
// returns the grouped model lists.
func fetchOfficialModels() (*officialModelsResponse, error) {
	req, err := http.NewRequest("GET", officialModelsURL, bytes.NewReader([]byte{}))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	cfg := getProxyConfig()
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("official models request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("official models API %d: %s", resp.StatusCode, truncate(readBody(resp), 300))
	}
	var data officialModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("official models decode: %w", err)
	}
	return &data, nil
}
