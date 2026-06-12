package hotrun

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const activeEnv = "SNOK_HOTRUN_ACTIVE"

type Link struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

type Registry struct {
	Links map[string]Link `json:"links"`
}

type Manager struct {
	home string
}

type LinkResult struct {
	Name     string
	Source   string
	LinkPath string
	BinDir   string
	PathHint string
}

func NewManager() (*Manager, error) {
	home := os.Getenv("SNOK_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		home = filepath.Join(userHome, ".snok")
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	return &Manager{home: abs}, nil
}

func NewManagerAt(home string) (*Manager, error) {
	if home == "" {
		return nil, errors.New("snok home is required")
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	return &Manager{home: abs}, nil
}

func (m *Manager) Home() string {
	return m.home
}

func (m *Manager) BinDir() string {
	return filepath.Join(m.home, "bin")
}

func (m *Manager) registryPath() string {
	return filepath.Join(m.home, "links.json")
}

func (m *Manager) cacheDir(name, hash string) string {
	return filepath.Join(m.home, "cache", "hotrun", name, hash)
}

func (m *Manager) ReadRegistry() (Registry, error) {
	data, err := os.ReadFile(m.registryPath())
	if errors.Is(err, os.ErrNotExist) {
		return Registry{Links: map[string]Link{}}, nil
	}
	if err != nil {
		return Registry{}, err
	}
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return Registry{}, err
	}
	if registry.Links == nil {
		registry.Links = map[string]Link{}
	}
	return registry, nil
}

func (m *Manager) WriteRegistry(registry Registry) error {
	if registry.Links == nil {
		registry.Links = map[string]Link{}
	}
	if err := os.MkdirAll(m.home, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(m.registryPath(), data, 0o644)
}

func (m *Manager) Link(name, source, launcher string) (LinkResult, error) {
	if err := validateName(name); err != nil {
		return LinkResult{}, err
	}
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return LinkResult{}, err
	}
	if info, err := os.Stat(sourceAbs); err != nil {
		return LinkResult{}, err
	} else if !info.IsDir() {
		return LinkResult{}, fmt.Errorf("source must be a directory: %s", sourceAbs)
	}
	if launcher == "" {
		launcher, err = os.Executable()
		if err != nil {
			return LinkResult{}, err
		}
	}
	launcher, err = filepath.Abs(launcher)
	if err != nil {
		return LinkResult{}, err
	}
	registry, err := m.ReadRegistry()
	if err != nil {
		return LinkResult{}, err
	}
	registry.Links[name] = Link{Name: name, Source: sourceAbs}
	if err := m.WriteRegistry(registry); err != nil {
		return LinkResult{}, err
	}
	if err := os.MkdirAll(m.BinDir(), 0o755); err != nil {
		return LinkResult{}, err
	}
	linkPath := filepath.Join(m.BinDir(), name)
	if err := os.Remove(linkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return LinkResult{}, err
	}
	if err := os.Symlink(launcher, linkPath); err != nil {
		return LinkResult{}, err
	}
	return LinkResult{
		Name:     name,
		Source:   sourceAbs,
		LinkPath: linkPath,
		BinDir:   m.BinDir(),
		PathHint: pathHint(m.BinDir()),
	}, nil
}

func (m *Manager) Unlink(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	registry, err := m.ReadRegistry()
	if err != nil {
		return err
	}
	delete(registry.Links, name)
	if err := m.WriteRegistry(registry); err != nil {
		return err
	}
	linkPath := filepath.Join(m.BinDir(), name)
	if err := os.Remove(linkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *Manager) Links() ([]Link, error) {
	registry, err := m.ReadRegistry()
	if err != nil {
		return nil, err
	}
	links := make([]Link, 0, len(registry.Links))
	for _, link := range registry.Links {
		links = append(links, link)
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].Name < links[j].Name
	})
	return links, nil
}

func (m *Manager) CachedBinary(name string) (string, bool, error) {
	registry, err := m.ReadRegistry()
	if err != nil {
		return "", false, err
	}
	link, ok := registry.Links[name]
	if !ok {
		return "", false, fmt.Errorf("unknown snok link %q", name)
	}
	hash, err := Fingerprint(link.Source)
	if err != nil {
		return "", false, err
	}
	binary := filepath.Join(m.cacheDir(name, hash), name)
	if executableExists(binary) {
		return binary, false, nil
	}
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		return "", false, err
	}
	if err := build(binary, link.Source); err != nil {
		return "", false, err
	}
	return binary, true, nil
}

func (m *Manager) Exec(name string, args []string) error {
	if os.Getenv(activeEnv) == "1" {
		return fmt.Errorf("refusing recursive hotrun invocation for %q", name)
	}
	binary, _, err := m.CachedBinary(name)
	if err != nil {
		return err
	}
	argv := append([]string{binary}, args...)
	env := append(os.Environ(), activeEnv+"=1")
	return syscall.Exec(binary, argv, env)
}

func Fingerprint(source string) (string, error) {
	root, err := moduleRoot(source)
	if err != nil {
		return "", err
	}
	var files []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			name := entry.Name()
			if name == "vendor" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		switch {
		case name == "go.mod", name == "go.sum":
			files = append(files, path)
		case strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go"):
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)
	hash := sha256.New()
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return "", err
		}
		if _, err := io.WriteString(hash, filepath.ToSlash(rel)+"\n"); err != nil {
			return "", err
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		if _, err := hash.Write(data); err != nil {
			return "", err
		}
		if _, err := io.WriteString(hash, "\n"); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil))[:24], nil
}

func moduleRoot(source string) (string, error) {
	current, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(current)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find go.mod for %s", source)
		}
		current = parent
	}
}

func build(binary, source string) error {
	root, err := moduleRoot(source)
	if err != nil {
		return err
	}
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, sourceAbs)
	if err != nil {
		return err
	}
	pkg := "."
	if rel != "." {
		pkg = "./" + filepath.ToSlash(rel)
	}
	cmd := exec.Command("go", "build", "-o", binary, pkg)
	cmd.Dir = root
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}
	return nil
}

func executableExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func validateName(name string) error {
	if name == "" || name == "." || name == ".." || name == "snok" {
		return fmt.Errorf("invalid link name %q", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid link name %q", name)
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			continue
		}
		return fmt.Errorf("invalid link name %q", name)
	}
	return nil
}

func pathHint(binDir string) string {
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if samePath(entry, binDir) {
			return ""
		}
	}
	return fmt.Sprintf("add %s to PATH to run linked commands directly", binDir)
}

func samePath(a, b string) bool {
	aAbs, aErr := filepath.Abs(a)
	bAbs, bErr := filepath.Abs(b)
	if aErr != nil || bErr != nil {
		return a == b
	}
	return aAbs == bAbs
}
