package minecraft

import (
	"context"
	"fmt"
)

var (
	cleanroomMetadataURL = "https://repo.cleanroommc.com/releases/com/cleanroommc/cleanroom/maven-metadata.xml"
	cleanroomMavenBase   = "https://repo.cleanroommc.com/releases/com/cleanroommc/cleanroom"
)

// FetchCleanroomBuilds lists Cleanroom releases. Cleanroom targets Minecraft
// 1.12.2 only.
func FetchCleanroomBuilds(ctx context.Context, hc *HTTPClient, mcVersion string) ([]LoaderBuild, error) {
	if mcVersion != "1.12.2" {
		return nil, nil
	}
	data, err := hc.Get(ctx, cleanroomMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("获取 Cleanroom 版本列表: %w", err)
	}
	versions, err := parseMavenMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("解析 Cleanroom 版本列表: %w", err)
	}
	var out []LoaderBuild
	for _, v := range versions {
		out = append(out, LoaderBuild{
			Version:   v,
			MCVersion: mcVersion,
			Kind:      "cleanroom",
			Stable:    false,
		})
	}
	sortLoaderBuilds(out)
	return out, nil
}
