package tui

import (
	"fmt"
	"strings"

	"github.com/MoonCC233/MCTL/internal/minecraft"
)

type modsInspectorData struct {
	GameDir string
	Mods    []minecraft.ModInfo
}

func (a *App) openModsInfo() {
	gameDir := a.worldGameDir()
	mods, err := minecraft.InspectMods(gameDir)
	if err != nil {
		a.openError("读取模组失败", err)
		return
	}
	items := make([]pickerItem, 0, len(mods))
	for _, m := range mods {
		var parts []string
		if m.Disabled {
			parts = append(parts, "已禁用")
		}
		if m.Err != "" {
			parts = append(parts, "读取失败")
		}
		if m.Loader != "" {
			parts = append(parts, m.Loader)
		}
		if m.ID != "" {
			id := m.ID
			if m.Version != "" {
				id += " " + m.Version
			}
			parts = append(parts, id)
		}
		if len(m.Nested) > 0 {
			parts = append(parts, fmt.Sprintf("含 %d 个内嵌 jar", len(m.Nested)))
		}
		if len(parts) == 0 {
			parts = append(parts, "无元数据")
		}
		items = append(items, pickerItem{Label: m.File, Desc: strings.Join(parts, " · "), Value: m.File})
	}
	a.modal = modalState{
		Kind:  modalMods,
		Title: fmt.Sprintf("模组信息 · %d", len(items)),
		Items: items,
		Data:  modsInspectorData{GameDir: gameDir, Mods: mods},
	}
}

func (a *App) openModDetail(m minecraft.ModInfo) {
	a.modal = modalState{Kind: modalModDetail, Title: "模组 · " + m.File, Data: m}
}

func modStatusLabel(m minecraft.ModInfo) string {
	if m.Disabled {
		return "已禁用"
	}
	return "已启用"
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
