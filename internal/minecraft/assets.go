package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AssetObject is one file in an asset index.
type AssetObject struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// AssetIndex is the objects map shipped with a version.
type AssetIndex struct {
	Objects map[string]AssetObject `json:"objects"`
}

func (i *Installer) installAssets(ctx context.Context, rv *ResolvedVersion) error {
	if rv.AssetIndex == nil || rv.AssetIndex.URL == "" {
		return nil
	}
	ref := rv.AssetIndex
	id := ref.ID
	if id == "" {
		id = rv.Assets
	}
	if id == "" {
		id = "legacy"
	}
	indexPath := filepath.Join(i.GameDir, "assets", "indexes", id+".json")
	if !fileMatches(indexPath, ref.Size) {
		i.logf("下载资源索引 %s", id)
		if err := i.HTTP.DownloadFile(ctx, ref.URL, indexPath, ref.SHA1, nil); err != nil {
			return fmt.Errorf("下载资源索引: %w", err)
		}
	}
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("读取资源索引: %w", err)
	}
	var idx AssetIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return fmt.Errorf("解析资源索引: %w", err)
	}

	var tasks []fileTask
	for name, obj := range idx.Objects {
		if obj.Hash == "" {
			continue
		}
		dest := filepath.Join(i.GameDir, "assets", "objects", obj.Hash[:2], obj.Hash)
		tasks = append(tasks, fileTask{
			URL:   "https://resources.download.minecraft.net/" + obj.Hash[:2] + "/" + obj.Hash,
			Dest:  dest,
			SHA1:  obj.Hash,
			Size:  obj.Size,
			Label: name,
		})
	}
	i.logf("检查 %d 个资源文件", len(tasks))
	if err := i.downloadAll(ctx, tasks, "下载资源文件"); err != nil {
		return err
	}

	if ref.Virtual || ref.MapToResources {
		if err := i.mapLegacyAssets(id, idx, ref.MapToResources); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) mapLegacyAssets(indexID string, idx AssetIndex, toResources bool) error {
	base := filepath.Join(i.GameDir, "assets", "virtual", indexID)
	if toResources {
		base = filepath.Join(i.GameDir, "resources")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	for name, obj := range idx.Objects {
		if obj.Hash == "" {
			continue
		}
		src := filepath.Join(i.GameDir, "assets", "objects", obj.Hash[:2], obj.Hash)
		dst := filepath.Join(base, filepath.FromSlash(name))
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Link(src, dst); err != nil {
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("映射旧版资源 %s: %w", name, err)
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
