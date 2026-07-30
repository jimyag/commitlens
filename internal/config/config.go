package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Repository struct {
	Owner     string `yaml:"owner"`
	Repo      string `yaml:"repo"`
	LocalPath string `yaml:"local_path"`
}

// IsOrganization reports whether this entry discovers repositories from an
// organization instead of naming one repository directly.
func (r Repository) IsOrganization() bool {
	return r.LocalPath == "" && r.Owner != "" && r.Repo == ""
}

// IsSyncable reports whether this entry identifies a concrete repository.
func (r Repository) IsSyncable() bool {
	return r.LocalPath != "" || (r.Owner != "" && r.Repo != "")
}

// ID returns a unique identifier for the repository.
func (r Repository) ID() string {
	if r.LocalPath != "" {
		return filepath.Base(r.LocalPath)
	}
	return r.Owner + "/" + r.Repo
}

type GitHub struct {
	Token string `yaml:"token"`
}

type Cache struct {
	Dir string `yaml:"dir"`
}

type Web struct {
	Port int `yaml:"port"`
}

type Reporting struct {
	Timezone string `yaml:"timezone"`
}

type Config struct {
	GitHub       GitHub              `yaml:"github"`
	Repositories []Repository        `yaml:"repositories"`
	Cache        Cache               `yaml:"cache"`
	Web          Web                 `yaml:"web"`
	Reporting    Reporting           `yaml:"reporting"`
	UserMap      map[string][]string `yaml:"userMap"`
	IgnoredUsers []string            `yaml:"ignoredUsers"`
	// DiscoveryRoots: Local directories to scan for Git repositories.
	DiscoveryRoots []string `yaml:"discoveryRoots"`
	// Language: UI language for TUI and CLI (en, zh). Empty means COMMITLENS_LANG / LANG.
	Language string `yaml:"language"`

	path string
	mu   sync.RWMutex
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	setDefaults(cfg)

	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.path = expandHome(path)
	cfg.Cache.Dir = expandHome(cfg.Cache.Dir)
	if cfg.Cache.Dir == "" {
		home, _ := os.UserHomeDir()
		cfg.Cache.Dir = filepath.Join(home, ".commitlens", "cache")
	}
	if cfg.Web.Port == 0 {
		cfg.Web.Port = 8080
	}
	if _, err := cfg.ReportingLocation(); err != nil {
		return nil, err
	}
	for i, root := range cfg.DiscoveryRoots {
		cfg.DiscoveryRoots[i] = expandHome(root)
	}
	return cfg, nil
}

// IdentitySettings returns copies of the configured aliases and ignored users.
func (c *Config) IdentitySettings() (map[string][]string, []string) {
	if c == nil {
		return nil, nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	userMap := make(map[string][]string, len(c.UserMap))
	for canonical, aliases := range c.UserMap {
		userMap[canonical] = append([]string(nil), aliases...)
	}
	return userMap, append([]string(nil), c.IgnoredUsers...)
}

// UpdateIdentitySettings persists only identity-related YAML fields and updates
// the in-memory configuration. Other fields and comments are left intact.
func (c *Config) UpdateIdentitySettings(userMap map[string][]string, ignoredUsers []string) error {
	if c == nil || c.path == "" {
		return fmt.Errorf("config path is unavailable")
	}

	normalizedMap := normalizeUserMap(userMap)
	normalizedIgnored := normalizeNames(ignoredUsers)

	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return err
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("config root must be a mapping")
	}

	root := document.Content[0]
	setMappingValue(root, "userMap", encodeUserMap(normalizedMap))
	setMappingValue(root, "ignoredUsers", encodeStringList(normalizedIgnored))

	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(c.path); statErr == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(c.path), ".commitlens-config-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	_ = tmp.Chmod(mode)

	encoder := yaml.NewEncoder(tmp)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		tmp.Close()
		return err
	}
	if err := encoder.Close(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, c.path); err != nil {
		return err
	}

	c.UserMap = normalizedMap
	c.IgnoredUsers = normalizedIgnored
	return nil
}

func normalizeUserMap(input map[string][]string) map[string][]string {
	result := make(map[string][]string, len(input))
	for canonical, aliases := range input {
		canonical = strings.TrimSpace(canonical)
		if canonical == "" {
			continue
		}
		normalized := normalizeNames(aliases)
		filtered := normalized[:0]
		for _, alias := range normalized {
			if !strings.EqualFold(alias, canonical) {
				filtered = append(filtered, alias)
			}
		}
		result[canonical] = filtered
	}
	return result
}

func normalizeNames(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))
	for _, value := range input {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result
}

func setMappingValue(root *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = value
			return
		}
	}
	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

func encodeUserMap(userMap map[string][]string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	keys := make([]string, 0, len(userMap))
	for canonical := range userMap {
		keys = append(keys, canonical)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})
	for _, canonical := range keys {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: canonical},
			encodeStringList(userMap[canonical]),
		)
	}
	return node
}

func encodeStringList(values []string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, value := range values {
		node.Content = append(node.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: value,
		})
	}
	return node
}

func (c *Config) ReportingLocation() (*time.Location, error) {
	if c == nil || c.Reporting.Timezone == "" || c.Reporting.Timezone == "Local" {
		return time.Local, nil
	}
	location, err := time.LoadLocation(c.Reporting.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid reporting timezone %q: %w", c.Reporting.Timezone, err)
	}
	return location, nil
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".commitlens", "config.yaml")
}

func setDefaults(cfg *Config) {
	home, _ := os.UserHomeDir()
	cfg.Cache.Dir = filepath.Join(home, ".commitlens", "cache")
	cfg.Web.Port = 8080
	cfg.Reporting.Timezone = "Local"
}
