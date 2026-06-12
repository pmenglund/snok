package snok

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

type NodeKind string

const (
	NodeGroup   NodeKind = "group"
	NodeCommand NodeKind = "command"
)

type Tree struct {
	Config Config
	Root   *Node
}

type Node struct {
	Name        string
	Kind        NodeKind
	Description string
	Aliases     []string
	Examples    []string
	Command     *CommandDefinition
	Group       *GroupDefinition
	Parent      *Node
	children    map[string]*Node
	order       []string
}

type RouteKind string

const (
	RouteCommand RouteKind = "command"
	RouteGroup   RouteKind = "group"
	RouteUnknown RouteKind = "unknown"
)

type RouteResult struct {
	Kind                RouteKind
	MatchedPath         []string
	RemainingArgs       []string
	HelpRequested       bool
	AttemptedPath       []string
	UnknownSegment      string
	AvailableChildNames []string
	Suggestions         []string
	Node                *Node
}

func NewTree(config Config) *Tree {
	rootGroup := DefineGroup(GroupDefinition{Description: config.Name})
	return &Tree{
		Config: DefineConfig(config),
		Root: &Node{
			Kind:     NodeGroup,
			Group:    &rootGroup,
			children: map[string]*Node{},
		},
	}
}

func (t *Tree) AddGroup(path []string, def GroupDefinition) error {
	if len(path) == 0 {
		group := DefineGroup(def)
		t.Root.Group = &group
		t.Root.Description = group.Description
		t.Root.Aliases = append([]string(nil), group.Aliases...)
		t.Root.Examples = append([]string(nil), group.Examples...)
		return nil
	}
	parent, err := t.ensureParent(path)
	if err != nil {
		return err
	}
	name := path[len(path)-1]
	if err := validateSegment(name); err != nil {
		return err
	}
	if existing := parent.children[name]; existing != nil && existing.Kind != NodeGroup {
		return fmt.Errorf("route %q is already a command", strings.Join(path, " "))
	}
	group := DefineGroup(def)
	node := parent.children[name]
	if node == nil {
		node = &Node{Name: name, Parent: parent, children: map[string]*Node{}}
		parent.children[name] = node
		parent.order = append(parent.order, name)
	}
	node.Kind = NodeGroup
	node.Group = &group
	node.Description = group.Description
	node.Aliases = append([]string(nil), group.Aliases...)
	node.Examples = append([]string(nil), group.Examples...)
	return nil
}

func (t *Tree) AddCommand(path []string, def CommandDefinition) error {
	if len(path) == 0 {
		cmd := DefineCommand(def)
		t.Root.Kind = NodeCommand
		t.Root.Command = &cmd
		t.Root.Description = cmd.Description
		t.Root.Aliases = nil
		t.Root.Examples = append([]string(nil), cmd.Examples...)
		return validateCommandDefinition(path, cmd, t.Config.Options)
	}
	parent, err := t.ensureParent(path)
	if err != nil {
		return err
	}
	name := path[len(path)-1]
	if err := validateSegment(name); err != nil {
		return err
	}
	cmd := DefineCommand(def)
	if err := validateCommandDefinition(path, cmd, t.Config.Options); err != nil {
		return err
	}
	node := parent.children[name]
	if node == nil {
		node = &Node{Name: name, Parent: parent, children: map[string]*Node{}}
		parent.children[name] = node
		parent.order = append(parent.order, name)
	} else if node.Kind == NodeGroup && node.Command == nil {
		// A group and command at the same logical path is valid only for executable groups.
	}
	node.Kind = NodeCommand
	node.Command = &cmd
	node.Description = cmd.Description
	node.Aliases = append([]string(nil), cmd.Aliases...)
	node.Examples = append([]string(nil), cmd.Examples...)
	return nil
}

func (t *Tree) ensureParent(parts []string) (*Node, error) {
	current := t.Root
	for _, segment := range parts[:len(parts)-1] {
		if err := validateSegment(segment); err != nil {
			return nil, err
		}
		child := current.children[segment]
		if child == nil {
			group := DefineGroup(GroupDefinition{})
			child = &Node{
				Name:     segment,
				Kind:     NodeGroup,
				Group:    &group,
				Parent:   current,
				children: map[string]*Node{},
			}
			current.children[segment] = child
			current.order = append(current.order, segment)
		}
		current = child
	}
	return current, nil
}

func (t *Tree) Resolve(argv []string) RouteResult {
	current := t.Root
	matched := []string{}
	for i, token := range argv {
		if token == "--" || strings.HasPrefix(token, "-") {
			return routeForNode(current, matched, argv[i:])
		}
		child := current.childFor(token)
		if child == nil {
			if current.Kind == NodeCommand {
				return routeForNode(current, matched, argv[i:])
			}
			available := current.childNames()
			return RouteResult{
				Kind:                RouteUnknown,
				MatchedPath:         append([]string(nil), matched...),
				AttemptedPath:       append(append([]string(nil), matched...), token),
				UnknownSegment:      token,
				AvailableChildNames: available,
				Suggestions:         current.suggestions(token),
				Node:                current,
			}
		}
		current = child
		matched = append(matched, child.Name)
	}
	return routeForNode(current, matched, nil)
}

func routeForNode(node *Node, matched []string, remaining []string) RouteResult {
	kind := RouteGroup
	if node.Kind == NodeCommand {
		kind = RouteCommand
	}
	return RouteResult{
		Kind:          kind,
		MatchedPath:   append([]string(nil), matched...),
		RemainingArgs: append([]string(nil), remaining...),
		HelpRequested: containsHelpBeforeTerminator(remaining),
		Node:          node,
	}
}

func containsHelpBeforeTerminator(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func (n *Node) childFor(token string) *Node {
	if n == nil {
		return nil
	}
	if child := n.children[token]; child != nil {
		return child
	}
	for _, name := range n.order {
		child := n.children[name]
		for _, alias := range child.Aliases {
			if alias == token {
				return child
			}
		}
	}
	return nil
}

func (n *Node) Children() []*Node {
	children := make([]*Node, 0, len(n.order))
	for _, name := range n.order {
		children = append(children, n.children[name])
	}
	return children
}

func (n *Node) Path() []string {
	var rev []string
	for current := n; current != nil && current.Parent != nil; current = current.Parent {
		rev = append(rev, current.Name)
	}
	out := make([]string, 0, len(rev))
	for i := len(rev) - 1; i >= 0; i-- {
		out = append(out, rev[i])
	}
	return out
}

func (n *Node) childNames() []string {
	out := append([]string(nil), n.order...)
	sort.Strings(out)
	return out
}

func (n *Node) suggestions(token string) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range n.order {
		child := n.children[name]
		if closeCommandToken(token, name) && !seen[name] {
			out = append(out, name)
			seen[name] = true
			continue
		}
		for _, alias := range child.Aliases {
			if closeCommandToken(token, alias) && !seen[name] {
				out = append(out, name)
				seen[name] = true
			}
		}
	}
	sort.Strings(out)
	return out
}

func closeCommandToken(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) == len(b) {
		for i := 0; i < len(a)-1; i++ {
			if a[i] == b[i+1] && a[i+1] == b[i] && a[:i] == b[:i] && a[i+2:] == b[i+2:] {
				return true
			}
		}
	}
	return levenshtein(a, b) <= 2
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current := make([]int, len(b)+1)
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			current[j] = min(current[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = current
	}
	return prev[len(b)]
}

type PublicTree struct {
	Commands [][]string
	Groups   [][]string
}

func DerivePublicTree(logicalUnits []string) PublicTree {
	commands := map[string][]string{}
	groups := map[string][]string{}
	for _, unit := range logicalUnits {
		parts := strings.Split(path.Clean(unit), "/")
		if len(parts) == 0 || parts[0] != "commands" {
			continue
		}
		public := parts[1:]
		if len(public) == 0 {
			continue
		}
		leaf := public[len(public)-1]
		if strings.HasPrefix(leaf, "_") || strings.HasSuffix(leaf, ".test") || strings.HasSuffix(leaf, ".spec") {
			continue
		}
		skip := false
		for _, segment := range public {
			if strings.HasPrefix(segment, "_") {
				skip = true
			}
		}
		if skip {
			continue
		}
		key := strings.Join(public, "\x00")
		commands[key] = append([]string(nil), public...)
		for i := 1; i < len(public); i++ {
			groups[strings.Join(public[:i], "\x00")] = append([]string(nil), public[:i]...)
		}
	}
	return PublicTree{
		Commands: sortedPaths(commands),
		Groups:   sortedPaths(groups),
	}
}

func sortedPaths(paths map[string][]string) [][]string {
	keys := make([]string, 0, len(paths))
	for key := range paths {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([][]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, paths[key])
	}
	return out
}
