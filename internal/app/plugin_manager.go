package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"haya-tab/pkg/logger"
	"haya-tab/pkg/store"

	"github.com/dop251/goja"
)

type PluginManifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Entry       string   `json:"entry"`
	Hooks       []string `json:"hooks"`
	Permissions []string `json:"permissions"`
}

type Plugin struct {
	Manifest PluginManifest
	Dir      string
	VM       *goja.Runtime
}

type PluginManager struct {
	plugins []Plugin
	logger  *logger.Logger
}

func NewPluginManager(logger *logger.Logger) *PluginManager {
	return &PluginManager{
		plugins: make([]Plugin, 0),
		logger:  logger,
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
		resp, err := http.Get(url)
		if err != nil {
			pm.logger.Error("Plugin fetch error: %v", err)
			return vm.ToValue(nil)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return vm.ToValue(string(body))
	})

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
	})
	pm.logger.Info("Loaded plugin: %s (%s)", manifest.Name, manifest.Version)
}

func (pm *PluginManager) EnhanceMetadata(tab *store.Tab) {
	for _, p := range pm.plugins {
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
