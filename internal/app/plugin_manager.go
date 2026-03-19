package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"haya-tab/pkg/logger"
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"

	"github.com/dop251/goja"
)

type PluginManifest struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	Entry          string            `json:"entry"`
	Hooks          []string          `json:"hooks"`
	Permissions    []string          `json:"permissions"`
	SettingsSchema map[string]string `json:"settingsSchema,omitempty"`
}

type PluginInfo struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	SettingsSchema map[string]string `json:"settingsSchema"`
	Config         map[string]string `json:"config"`
	Enabled        bool              `json:"enabled"`
}

type Plugin struct {
	Manifest PluginManifest
	Dir      string
	VM       *goja.Runtime
	Enabled  bool
	Config   map[string]string
}

type PluginManager struct {
	plugins    []Plugin
	logger     *logger.Logger
	mu         sync.RWMutex
	httpClient *http.Client
}

// NewPluginManager creates a new PluginManager.
// All plugin configuration is stored next to the plugin itself in a
// plugin-local config.json file inside the plugin directory.
func NewPluginManager(logger *logger.Logger) *PluginManager {
	return &PluginManager{
		plugins: make([]Plugin, 0),
		logger:  logger,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (pm *PluginManager) Init(pluginsDir string) error {
	pm.logger.Info("Initializing PluginManager from: %s", pluginsDir)
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(pluginsDir, entry.Name())
		pm.loadPlugin(pluginDir)
	}

	return nil
}

// StartSyncRun should be called at the beginning of each sync run.
// It resets per-run counters in all plugins so that they can enforce
// limits such as "maxRequestsPerRun" within a single sync invocation.
func (pm *PluginManager) StartSyncRun() {
	pm.logger.Info("PluginManager: starting new sync run, resetting per-run counters")
	for _, p := range pm.plugins {
		if p.VM == nil {
			continue
		}
		// Ignore error; failing to set the counter should not break sync.
		_ = p.VM.Set("requestCountThisRun", 0)
	}
}

// loadPluginConfig loads a plugin-local configuration from "config.json"
// inside the plugin directory. The file is expected to be a simple
// JSON object with string values, e.g.:
//
//	{
//	  "baseUrl": "https://api.openai.com/v1",
//	  "apiKey": "sk-...",
//	  "model": "gpt-4o-mini"
//	}
func (pm *PluginManager) loadPluginConfig(pluginDir string) map[string]string {
	cfgPath := filepath.Join(pluginDir, "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		// Missing config is not an error; plugins may choose to run with defaults.
		return map[string]string{}
	}
	var cfg map[string]string
	if err := json.Unmarshal(data, &cfg); err != nil {
		pm.logger.Error("Failed to parse plugin config %s: %v", cfgPath, err)
		return map[string]string{}
	}
	return cfg
}

func (pm *PluginManager) loadPlugin(pluginDir string) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		pm.logger.Info("Plugin manifest not found in %s: %v", pluginDir, err)
		return
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		pm.logger.Error("Failed to parse plugin manifest in %s: %v", pluginDir, err)
		return
	}

	entryPath := filepath.Join(pluginDir, manifest.Entry)
	script, err := os.ReadFile(entryPath)
	if err != nil {
		pm.logger.Error("Failed to read plugin entry script %s: %v", entryPath, err)
		return
	}

	vm := goja.New()

	// Expose logger
	vm.Set("log", func(msg string) {
		pm.logger.Info("[Plugin %s] %s", manifest.Name, msg)
	})

	// Expose fetch (simple implementation)
	vm.Set("fetch", func(url string) goja.Value {
		resp, err := pm.httpClient.Get(url)
		if err != nil {
			pm.logger.Error("Plugin fetch error: %v", err)
			return vm.ToValue(nil)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return vm.ToValue(string(body))
	})

	// Expose httpRequest for more advanced use cases (e.g. AI APIs).
	// Request shape from JS:
	// {
	//   method: "GET" | "POST" | ...,
	//   url: "...",
	//   headers: { "Header-Name": "value" },
	//   body: "raw body as string"
	// }
	// Response shape:
	// {
	//   status: number,
	//   body: string
	// }
	vm.Set("httpRequest", func(req map[string]interface{}) map[string]interface{} {
		method, _ := req["method"].(string)
		if method == "" {
			method = http.MethodGet
		}
		url, ok := req["url"].(string)
		if !ok || url == "" {
			return map[string]interface{}{
				"status": 0,
				"body":   "",
				"error":  "missing url",
			}
		}

		bodyStr, _ := req["body"].(string)

		var headers map[string]string
		if rawHeaders, ok := req["headers"].(map[string]interface{}); ok {
			headers = make(map[string]string, len(rawHeaders))
			for k, v := range rawHeaders {
				if sv, ok := v.(string); ok {
					headers[k] = sv
				}
			}
		}

		httpReq, err := http.NewRequest(method, url, io.NopCloser(strings.NewReader(bodyStr)))
		if err != nil {
			pm.logger.Error("Plugin httpRequest build error: %v", err)
			return map[string]interface{}{
				"status": 0,
				"body":   "",
				"error":  err.Error(),
			}
		}
		for k, v := range headers {
			httpReq.Header.Set(k, v)
		}

		resp, err := pm.httpClient.Do(httpReq)
		if err != nil {
			pm.logger.Error("Plugin httpRequest error: %v", err)
			return map[string]interface{}{
				"status": 0,
				"body":   "",
				"error":  err.Error(),
			}
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		return map[string]interface{}{
			"status": resp.StatusCode,
			"body":   string(respBody),
		}
	})

	// Load configuration for this plugin (if any) and expose as global "config".
	cfg := pm.loadPluginConfig(pluginDir)
	enabled := true
	if val, ok := cfg["__enabled"]; ok && val == "false" {
		enabled = false
	}
	// Do not expose __enabled to JS to keep it clean
	delete(cfg, "__enabled")

	vm.Set("config", cfg)

	// Inject module.exports = {} setup so script can export its functions
	_, err = vm.RunString(`var module = { exports: {} };`)
	if err != nil {
		pm.logger.Error("Failed to initialize module exports for plugin %s: %v", manifest.Name, err)
		return
	}

	_, err = vm.RunScript(manifest.Entry, string(script))
	if err != nil {
		pm.logger.Error("Failed to run script for plugin %s: %v", manifest.Name, err)
		return
	}

	pm.plugins = append(pm.plugins, Plugin{
		Manifest: manifest,
		Dir:      pluginDir,
		VM:       vm,
		Enabled:  enabled,
		Config:   cfg,
	})
	pm.logger.Info("Loaded plugin: %s (%s)", manifest.Name, manifest.Version)
}

func (pm *PluginManager) GetPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var infos []PluginInfo
	for _, p := range pm.plugins {
		infos = append(infos, PluginInfo{
			ID:             p.Manifest.ID,
			Name:           p.Manifest.Name,
			Version:        p.Manifest.Version,
			SettingsSchema: p.Manifest.SettingsSchema,
			Config:         p.Config,
			Enabled:        p.Enabled,
		})
	}
	return infos
}

func (pm *PluginManager) HasPlugins() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.plugins) > 0
}

func (pm *PluginManager) UpdatePluginConfig(id string, config map[string]string, enabled bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var targetPlugin *Plugin
	for i := range pm.plugins {
		if pm.plugins[i].Manifest.ID == id {
			targetPlugin = &pm.plugins[i]
			break
		}
	}

	if targetPlugin == nil {
		return fmt.Errorf("plugin %s not found", id)
	}

	targetPlugin.Enabled = enabled
	if config == nil {
		targetPlugin.Config = make(map[string]string)
	} else {
		targetPlugin.Config = config
	}

	// Save to config.json
	saveMap := make(map[string]string)
	for k, v := range targetPlugin.Config {
		saveMap[k] = v
	}
	if enabled {
		saveMap["__enabled"] = "true"
	} else {
		saveMap["__enabled"] = "false"
	}

	cfgPath := filepath.Join(targetPlugin.Dir, "config.json")
	data, err := json.MarshalIndent(saveMap, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return err
	}

	// Update VM config
	if targetPlugin.VM != nil {
		targetPlugin.VM.Set("config", targetPlugin.Config)
	}

	return nil
}

func (pm *PluginManager) EnhanceMetadata(tab *store.Tab) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, p := range pm.plugins {
		if !p.Enabled {
			continue
		}

		// Check if it has metadata hook
		hasHook := false
		for _, h := range p.Manifest.Hooks {
			if h == "metadata" {
				hasHook = true
				break
			}
		}
		if !hasHook {
			continue
		}

		exportsVal := p.VM.Get("module").ToObject(p.VM).Get("exports")
		if exportsVal == nil || goja.IsUndefined(exportsVal) {
			continue
		}
		exports := exportsVal.ToObject(p.VM)
		enhanceMetadataFn := exports.Get("enhanceMetadata")

		if enhanceMetadataFn == nil || goja.IsUndefined(enhanceMetadataFn) {
			continue
		}

		fn, ok := goja.AssertFunction(enhanceMetadataFn)
		if !ok {
			pm.logger.Error("Plugin %s enhanceMetadata is not a function", p.Manifest.Name)
			continue
		}

		// Convert tab to JS object (using JSON serialization as bridge)
		tabJSON, _ := json.Marshal(tab)
		var tabMap map[string]interface{}
		json.Unmarshal(tabJSON, &tabMap)

		jsTab := p.VM.ToValue(tabMap)

		res, err := fn(goja.Undefined(), jsTab)
		if err != nil {
			pm.logger.Error("Plugin %s enhanceMetadata error: %v", p.Manifest.Name, err)
			continue
		}

		// Read back
		resExport := res.Export()
		if resMap, ok := resExport.(map[string]interface{}); ok {
			// Convert map back to Tab
			resJSON, _ := json.Marshal(resMap)
			json.Unmarshal(resJSON, tab)
		}
	}
}

func (pm *PluginManager) DownloadCover(artist, album, title, country, lang, dstPath string) error {
	pm.mu.RLock()
	plugins := make([]Plugin, len(pm.plugins))
	copy(plugins, pm.plugins)
	pm.mu.RUnlock()

	for _, p := range plugins {
		if !p.Enabled {
			continue
		}

		hasHook := false
		for _, h := range p.Manifest.Hooks {
			if h == "cover" {
				hasHook = true
				break
			}
		}
		if !hasHook {
			continue
		}

		exportsVal := p.VM.Get("module").ToObject(p.VM).Get("exports")
		if exportsVal == nil || goja.IsUndefined(exportsVal) {
			continue
		}
		exports := exportsVal.ToObject(p.VM)
		getCoverUrlFn := exports.Get("getCoverUrl")

		if getCoverUrlFn == nil || goja.IsUndefined(getCoverUrlFn) {
			continue
		}

		fn, ok := goja.AssertFunction(getCoverUrlFn)
		if !ok {
			pm.logger.Error("Plugin %s getCoverUrl is not a function", p.Manifest.Name)
			continue
		}

		res, err := fn(goja.Undefined(), p.VM.ToValue(artist), p.VM.ToValue(album), p.VM.ToValue(title), p.VM.ToValue(country), p.VM.ToValue(lang))
		if err != nil {
			pm.logger.Error("Plugin %s getCoverUrl error: %v", p.Manifest.Name, err)
			continue
		}

		if res == nil || goja.IsUndefined(res) || goja.IsNull(res) {
			continue // plugin didn't find anything
		}

		urlStr := res.String()
		if urlStr != "" {
			pm.logger.Info("Plugin %s provided cover URL: %s", p.Manifest.Name, urlStr)
			// Download the URL
			err := pm.downloadFile(urlStr, dstPath)
			if err == nil {
				return nil // Success
			}
			pm.logger.Error("Plugin %s failed to download cover from %s: %v", p.Manifest.Name, urlStr, err)
		}
	}

	// Fallback to iTunes
	return metadata.DownloadCover(artist, album, title, country, lang, dstPath)
}

func (pm *PluginManager) downloadFile(urlStr, dstPath string) error {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}
	// Add common User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := pm.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
