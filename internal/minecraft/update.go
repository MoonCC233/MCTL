package minecraft

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"strings"
)

// releasesFeedBase serves the public release feeds.
var releasesFeedBase = "https://github.com"

// ReleaseInfo is the subset of a published release used by the update check.
type ReleaseInfo struct {
	TagName     string
	Name        string
	HTMLURL     string
	PublishedAt string
	Body        string
}

type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Updated string `xml:"updated"`
	Title   string `xml:"title"`
	Content string `xml:"content"`
	Link    struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
}

// FetchLatestRelease reads the newest release from a repository's public Atom
// feed. The feed needs no API token and avoids the GitHub API rate limit.
func FetchLatestRelease(ctx context.Context, hc *HTTPClient, owner, repo string) (*ReleaseInfo, error) {
	feed, err := fetchReleaseFeed(ctx, hc, owner, repo)
	if err != nil {
		return nil, err
	}
	if len(feed.Entries) == 0 {
		return nil, fmt.Errorf("没有找到发布版本")
	}
	info := releaseInfoFromEntry(feed.Entries[0])
	if info.TagName == "" {
		return nil, fmt.Errorf("最新版本缺少标签")
	}
	return &info, nil
}

// FetchRecentReleases returns the newest releases from the public feed.
func FetchRecentReleases(ctx context.Context, hc *HTTPClient, owner, repo string, limit int) ([]ReleaseInfo, error) {
	feed, err := fetchReleaseFeed(ctx, hc, owner, repo)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(feed.Entries) {
		limit = len(feed.Entries)
	}
	out := make([]ReleaseInfo, 0, limit)
	for _, entry := range feed.Entries[:limit] {
		out = append(out, releaseInfoFromEntry(entry))
	}
	return out, nil
}

func fetchReleaseFeed(ctx context.Context, hc *HTTPClient, owner, repo string) (*atomFeed, error) {
	if hc == nil {
		return nil, fmt.Errorf("缺少 HTTP 客户端")
	}
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("缺少仓库信息")
	}
	feedURL := fmt.Sprintf("%s/%s/%s/releases.atom", strings.TrimRight(releasesFeedBase, "/"), owner, repo)
	data, err := hc.Get(ctx, feedURL)
	if err != nil {
		return nil, fmt.Errorf("获取最新版本: %w", err)
	}
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("解析版本信息: %w", err)
	}
	return &feed, nil
}

func releaseInfoFromEntry(entry atomEntry) ReleaseInfo {
	tag := ""
	if href, err := url.Parse(entry.Link.Href); err == nil {
		tag = strings.TrimSpace(path.Base(href.Path))
	}
	if tag == "" {
		tag = strings.TrimSpace(entry.Title)
	}
	return ReleaseInfo{
		TagName:     tag,
		Name:        strings.TrimSpace(entry.Title),
		HTMLURL:     entry.Link.Href,
		PublishedAt: entry.Updated,
		Body:        entry.Content,
	}
}

// CompareVersions compares dotted version strings, ignoring a leading v.
func CompareVersions(a, b string) int {
	return compareVersions(a, b)
}

// IsNewerVersion reports whether latest is newer than current.
func IsNewerVersion(latest, current string) bool {
	return compareVersions(strings.TrimPrefix(strings.TrimSpace(latest), "v"), strings.TrimPrefix(strings.TrimSpace(current), "v")) > 0
}
