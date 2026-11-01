package minecraft

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Download describes one downloadable artifact.
type Download struct {
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

// AssetIndexRef points at a version's asset index.
type AssetIndexRef struct {
	ID             string `json:"id"`
	SHA1           string `json:"sha1"`
	Size           int64  `json:"size"`
	TotalSize      int64  `json:"totalSize"`
	URL            string `json:"url"`
	Virtual        bool   `json:"virtual"`
	MapToResources bool   `json:"map_to_resources"`
}

// OS names follow Mojang's vocabulary.
func osName() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "osx"
	default:
		return "linux"
	}
}

func osArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "386":
		return "x86"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}

func nativeArchBits() string {
	if strconv.IntSize == 64 {
		return "64"
	}
	return "32"
}

// Rule is a conditional allow/deny entry.
type Rule struct {
	Action   string          `json:"action"`
	OS       *OSRule         `json:"os,omitempty"`
	Features map[string]bool `json:"features,omitempty"`
}

// OSRule constrains a rule to an operating system.
type OSRule struct {
	Name    string `json:"name,omitempty"`
	Arch    string `json:"arch,omitempty"`
	Version string `json:"version,omitempty"`
}

func (r Rule) matches(features map[string]bool) bool {
	if r.OS != nil {
		if r.OS.Name != "" && r.OS.Name != osName() {
			return false
		}
		if r.OS.Arch != "" && r.OS.Arch != osArch() {
			return false
		}
		// OS version predicates are rare and almost always Windows-10 gates.
		// Treat them as satisfied rather than excluding required libraries.
	}
	for k, v := range r.Features {
		if features[k] != v {
			return false
		}
	}
	return true
}

// RulesAllowed evaluates a rule list (last matching rule wins).
func RulesAllowed(rules []Rule, features map[string]bool) bool {
	if len(rules) == 0 {
		return true
	}
	allowed := false
	for _, r := range rules {
		if r.matches(features) {
			allowed = r.Action == "allow"
		}
	}
	return allowed
}

// Arguments holds conditional JVM or game arguments.
type Arguments struct {
	Game []Argument `json:"game"`
	JVM  []Argument `json:"jvm"`
}

// Argument is either a plain string or a conditional object.
type Argument struct {
	Values []string
	Rules  []Rule
}

// UnmarshalJSON accepts both supported argument encodings.
func (a *Argument) UnmarshalJSON(data []byte) error {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		a.Values = []string{s}
		return nil
	}
	if data[0] == '[' {
		var ss []string
		if err := json.Unmarshal(data, &ss); err != nil {
			return err
		}
		a.Values = ss
		return nil
	}
	var obj struct {
		Value json.RawMessage `json:"value"`
		Rules []Rule          `json:"rules"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	a.Rules = obj.Rules
	v := strings.TrimSpace(string(obj.Value))
	if len(v) == 0 {
		return nil
	}
	if v[0] == '[' {
		return json.Unmarshal(obj.Value, &a.Values)
	}
	var s string
	if err := json.Unmarshal(obj.Value, &s); err != nil {
		return err
	}
	a.Values = []string{s}
	return nil
}

// Extract configures native archive extraction.
type Extract struct {
	Exclude []string `json:"exclude"`
}

// Library is a Maven library entry.
type Library struct {
	Name      string            `json:"name"`
	Downloads *LibraryDownloads `json:"downloads,omitempty"`
	Natives   map[string]string `json:"natives,omitempty"`
	Extract   *Extract          `json:"extract,omitempty"`
	Rules     []Rule            `json:"rules,omitempty"`
	URL       string            `json:"url,omitempty"`
}

// LibraryDownloads carries artifacts and platform classifiers.
type LibraryDownloads struct {
	Artifact    *Download           `json:"artifact,omitempty"`
	Classifiers map[string]Download `json:"classifiers,omitempty"`
}

// JavaVersion is the Java runtime requirement.
type JavaVersion struct {
	Component    string `json:"component"`
	MajorVersion int    `json:"majorVersion"`
}

// Logging configures the log4j configuration shipped with modern versions.
type Logging struct {
	Client *LoggingClient `json:"client,omitempty"`
}

// LoggingClient is one logging configuration file.
type LoggingClient struct {
	Argument string   `json:"argument"`
	File     Download `json:"file"`
	Type     string   `json:"type"`
}

// VersionJSON mirrors the official version document.
type VersionJSON struct {
	ID                 string              `json:"id"`
	InheritsFrom       string              `json:"inheritsFrom,omitempty"`
	Type               string              `json:"type"`
	MainClass          string              `json:"mainClass"`
	Assets             string              `json:"assets"`
	AssetIndex         *AssetIndexRef      `json:"assetIndex,omitempty"`
	Downloads          map[string]Download `json:"downloads,omitempty"`
	Libraries          []Library           `json:"libraries"`
	JavaVersion        *JavaVersion        `json:"javaVersion,omitempty"`
	Arguments          *Arguments          `json:"arguments,omitempty"`
	MinecraftArguments string              `json:"minecraftArguments,omitempty"`
	Logging            *Logging            `json:"logging,omitempty"`
	Jar                string              `json:"jar,omitempty"`
	MinecraftVersion   string              `json:"_minecraftVersion,omitempty"`
	ReleaseTime        string              `json:"releaseTime"`
	Time               string              `json:"time"`
	ComplianceLevel    int                 `json:"complianceLevel"`
}

// ResolvedVersion is a fully merged version ready to install or launch.
type ResolvedVersion struct {
	ID                 string
	Chain              []string
	Type               string
	MainClass          string
	Assets             string
	AssetIndex         *AssetIndexRef
	Libraries          []Library
	JVMArgs            []Argument
	GameArgs           []Argument
	MinecraftArguments string
	JavaVersion        *JavaVersion
	Logging            *Logging
	Downloads          map[string]Download
	JarID              string
	JarPath            string
	JSONPath           string
	ReleaseTime        string
	InheritsFrom       string
}

// VersionPath returns the directory for an installed version.
func VersionPath(gameDir, id string) string {
	return filepath.Join(gameDir, "versions", id)
}

// VersionJSONPath returns the version document path.
func VersionJSONPath(gameDir, id string) string {
	return filepath.Join(VersionPath(gameDir, id), id+".json")
}

// ReadVersionJSON parses a version document from disk.
func ReadVersionJSON(p string) (*VersionJSON, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var v VersionJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("解析 %s: %w", filepath.Base(p), err)
	}
	if v.ID == "" {
		v.ID = strings.TrimSuffix(filepath.Base(p), ".json")
	}
	return &v, nil
}

// ResolveVersion loads a version and all parents, merging them.
func ResolveVersion(gameDir, id string) (*ResolvedVersion, error) {
	var chain []*VersionJSON
	seen := map[string]bool{}
	cur := id
	for cur != "" {
		if seen[cur] {
			return nil, fmt.Errorf("版本继承出现循环: %s", cur)
		}
		seen[cur] = true
		p := VersionJSONPath(gameDir, cur)
		v, err := ReadVersionJSON(p)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("版本 %s 未安装或缺少父版本 %s", id, cur)
			}
			return nil, err
		}
		chain = append(chain, v)
		cur = v.InheritsFrom
	}

	base := *chain[len(chain)-1]
	base.Libraries = append([]Library(nil), base.Libraries...)
	var mergedArgs Arguments
	if base.Arguments != nil {
		mergedArgs = *base.Arguments
		mergedArgs.Game = append([]Argument(nil), base.Arguments.Game...)
		mergedArgs.JVM = append([]Argument(nil), base.Arguments.JVM...)
	}
	for i := len(chain) - 2; i >= 0; i-- {
		child := chain[i]
		if child.Type != "" {
			base.Type = child.Type
		}
		if child.MainClass != "" {
			base.MainClass = child.MainClass
		}
		if child.Assets != "" {
			base.Assets = child.Assets
		}
		if child.AssetIndex != nil {
			base.AssetIndex = child.AssetIndex
		}
		if child.JavaVersion != nil {
			base.JavaVersion = child.JavaVersion
		}
		if child.Logging != nil {
			base.Logging = child.Logging
		}
		if child.MinecraftArguments != "" {
			base.MinecraftArguments = child.MinecraftArguments
		}
		if child.Arguments != nil {
			mergedArgs.Game = append(mergedArgs.Game, child.Arguments.Game...)
			mergedArgs.JVM = append(mergedArgs.JVM, child.Arguments.JVM...)
		}
		if child.ReleaseTime != "" {
			base.ReleaseTime = child.ReleaseTime
		}
		base.Libraries = append(base.Libraries, child.Libraries...)
	}
	if len(mergedArgs.Game) > 0 || len(mergedArgs.JVM) > 0 {
		base.Arguments = &mergedArgs
	}

	libraries := dedupeLibraries(base.Libraries)

	jarID := ""
	for _, v := range chain {
		if v.Jar != "" {
			jarID = v.Jar
			break
		}
		if d, ok := v.Downloads["client"]; ok && d.URL != "" {
			jarID = v.ID
			break
		}
	}
	if jarID == "" {
		jarID = id
	}

	r := &ResolvedVersion{
		ID:                 id,
		Chain:              make([]string, 0, len(chain)),
		Type:               base.Type,
		MainClass:          base.MainClass,
		Assets:             base.Assets,
		AssetIndex:         base.AssetIndex,
		Libraries:          libraries,
		MinecraftArguments: base.MinecraftArguments,
		JavaVersion:        base.JavaVersion,
		Logging:            base.Logging,
		Downloads:          base.Downloads,
		JarID:              jarID,
		JarPath:            filepath.Join(VersionPath(gameDir, jarID), jarID+".jar"),
		JSONPath:           VersionJSONPath(gameDir, id),
		ReleaseTime:        base.ReleaseTime,
		InheritsFrom:       chain[0].InheritsFrom,
	}
	for _, v := range chain {
		r.Chain = append(r.Chain, v.ID)
	}
	if base.Arguments != nil {
		r.GameArgs = base.Arguments.Game
		r.JVMArgs = base.Arguments.JVM
	}
	if r.AssetIndex != nil && r.Assets == "" {
		r.Assets = r.AssetIndex.ID
	}
	return r, nil
}

func dedupeLibraries(in []Library) []Library {
	out := make([]Library, 0, len(in))
	index := map[string]int{}
	for _, lib := range in {
		key := libKey(lib.Name)
		if i, ok := index[key]; ok {
			out[i] = lib
			continue
		}
		index[key] = len(out)
		out = append(out, lib)
	}
	return out
}

func libKey(name string) string {
	c, err := ParseMaven(name)
	if err != nil {
		return name
	}
	return c.Group + ":" + c.Artifact + ":" + c.Classifier
}

// MavenCoord is a parsed Maven coordinate.
type MavenCoord struct {
	Group      string
	Artifact   string
	Version    string
	Classifier string
	Ext        string
}

// ParseMaven parses group:artifact:version[:classifier][@ext].
func ParseMaven(name string) (MavenCoord, error) {
	var c MavenCoord
	if name == "" {
		return c, fmt.Errorf("空 Maven 坐标")
	}
	ext := "jar"
	if i := strings.LastIndex(name, "@"); i >= 0 {
		ext = name[i+1:]
		name = name[:i]
	}
	parts := strings.Split(name, ":")
	if len(parts) < 3 {
		return c, fmt.Errorf("无效 Maven 坐标: %s", name)
	}
	c.Group, c.Artifact, c.Version = parts[0], parts[1], parts[2]
	if len(parts) >= 4 {
		c.Classifier = parts[3]
	}
	c.Ext = ext
	return c, nil
}

// Path returns the Maven repository relative path.
func (c MavenCoord) Path() string {
	name := c.Artifact + "-" + c.Version
	if c.Classifier != "" {
		name += "-" + c.Classifier
	}
	return path.Join(strings.ReplaceAll(c.Group, ".", "/"), c.Artifact, c.Version, name+"."+c.Ext)
}

// URL joins a repository base with the coordinate path.
func (c MavenCoord) URL(base string) string {
	if base == "" {
		base = "https://libraries.minecraft.net/"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + c.Path()
}

// Allowed reports whether the library applies on this platform.
func (l Library) Allowed(features map[string]bool) bool {
	return RulesAllowed(l.Rules, features)
}

// Coordinate parses the library's Maven name.
func (l Library) Coordinate() (MavenCoord, error) {
	return ParseMaven(l.Name)
}

// Artifact returns the main jar, synthesizing Maven data when needed.
func (l Library) Artifact() (*Download, error) {
	if l.Downloads != nil && l.Downloads.Artifact != nil {
		d := *l.Downloads.Artifact
		if d.Path == "" {
			if c, err := l.Coordinate(); err == nil {
				d.Path = c.Path()
			}
		}
		if d.URL == "" {
			if c, err := l.Coordinate(); err == nil {
				d.URL = c.URL(l.URL)
			}
		}
		if d.Path == "" {
			return nil, fmt.Errorf("库 %s 缺少路径", l.Name)
		}
		return &d, nil
	}
	c, err := l.Coordinate()
	if err != nil {
		return nil, err
	}
	return &Download{Path: c.Path(), URL: c.URL(l.URL)}, nil
}

// NativeClassifier returns the platform native classifier for this library.
func (l Library) NativeClassifier() (string, bool) {
	if len(l.Natives) == 0 {
		return "", false
	}
	tmpl, ok := l.Natives[osName()]
	if !ok {
		return "", false
	}
	return strings.ReplaceAll(tmpl, "${arch}", nativeArchBits()), true
}

// NativeArtifact returns the native jar for this platform.
func (l Library) NativeArtifact() (*Download, string, error) {
	classifier, ok := l.NativeClassifier()
	if !ok {
		return nil, "", nil
	}
	if l.Downloads != nil {
		if d, ok := l.Downloads.Classifiers[classifier]; ok {
			if d.Path == "" {
				if c, err := l.Coordinate(); err == nil {
					c.Classifier = classifier
					d.Path = c.Path()
				}
			}
			if d.URL == "" {
				if c, err := l.Coordinate(); err == nil {
					c.Classifier = classifier
					d.URL = c.URL(l.URL)
				}
			}
			return &d, classifier, nil
		}
	}
	c, err := l.Coordinate()
	if err != nil {
		return nil, "", err
	}
	c.Classifier = classifier
	return &Download{Path: c.Path(), URL: c.URL(l.URL)}, classifier, nil
}

// Features builds the rule feature map for a launch.
func Features(width, height int) map[string]bool {
	return map[string]bool{
		"has_custom_resolution":   width > 0 && height > 0,
		"is_demo_user":            false,
		"has_quick_plays_support": false,
	}
}

// GameArguments flattens conditional game arguments.
func (r *ResolvedVersion) GameArguments(features map[string]bool) []string {
	var out []string
	for _, a := range r.GameArgs {
		if RulesAllowed(a.Rules, features) {
			out = append(out, a.Values...)
		}
	}
	return out
}

// JVMArguments flattens conditional JVM arguments.
func (r *ResolvedVersion) JVMArguments(features map[string]bool) []string {
	var out []string
	for _, a := range r.JVMArgs {
		if RulesAllowed(a.Rules, features) {
			out = append(out, a.Values...)
		}
	}
	return out
}

// LibrariesFor returns allowed libraries, optionally only those with natives.
func (r *ResolvedVersion) LibrariesFor(features map[string]bool) []Library {
	var out []Library
	for _, l := range r.Libraries {
		if l.Allowed(features) {
			out = append(out, l)
		}
	}
	return out
}

// ClientDownloads returns client jars across the inheritance chain.
func ClientDownloads(gameDir, id string) (map[string]Download, error) {
	out := map[string]Download{}
	seen := map[string]bool{}
	cur := id
	for cur != "" {
		if seen[cur] {
			break
		}
		seen[cur] = true
		v, err := ReadVersionJSON(VersionJSONPath(gameDir, cur))
		if err != nil {
			return nil, err
		}
		if d, ok := v.Downloads["client"]; ok && d.URL != "" {
			out[v.ID] = d
		}
		cur = v.InheritsFrom
	}
	return out, nil
}

// VersionReleaseTime parses the version's release timestamp.
func VersionReleaseTime(gameDir, id string) string {
	v, err := ReadVersionJSON(VersionJSONPath(gameDir, id))
	if err != nil {
		return ""
	}
	return v.ReleaseTime
}

// SortVersions orders installed versions newest first.
func SortVersions(ids []string) {
	sort.SliceStable(ids, func(i, j int) bool { return ids[i] > ids[j] })
}

var versionRe = regexp.MustCompile(`^(\d+\.\d+(?:\.\d+)?)$`)

// LooksLikeMCVersion reports whether id is a plain Minecraft version.
func LooksLikeMCVersion(id string) bool { return versionRe.MatchString(id) }
