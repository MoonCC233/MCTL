package tui

import (
	"fmt"
	"strings"

	"github.com/MoonCC233/MCTL/internal/minecraft"
)

// openJavaPicker shows every detected Java runtime so the user can pick one
// with the mouse. Selecting an entry stores it as the global Java path; the
// first entry clears the override and returns to automatic detection.
func (a *App) openJavaPicker(list []minecraft.JavaInfo) {
	if len(list) == 0 {
		a.setToast("未检测到 Java，请安装 JDK 或手动填写路径", 2)
		return
	}
	cfg := a.store.Snapshot()
	items := []pickerItem{{
		Label: "自动检测",
		Desc:  "清除指定路径，按游戏版本自动选择",
		Value: "",
	}}
	for _, j := range list {
		label := fmt.Sprintf("Java %d", j.Major)
		if j.Vendor != "" {
			label += " · " + shortVendor(j.Vendor)
		}
		if j.Version != "" {
			label += " · " + j.Version
		}
		desc := j.Path
		if strings.EqualFold(j.Path, cfg.JavaPath) {
			label = "★ " + label
		}
		items = append(items, pickerItem{Label: label, Desc: desc, Value: j.Path})
	}
	a.modal = modalState{
		Kind:     modalPicker,
		Title:    fmt.Sprintf("已检测到的 Java · %d", len(list)),
		Items:    items,
		Selected: 0,
		Action:   "select-java",
	}
}

func shortVendor(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.Index(v, "("); i > 0 {
		v = strings.TrimSpace(v[:i])
	}
	if len([]rune(v)) > 22 {
		v = string([]rune(v)[:22]) + "…"
	}
	return v
}
