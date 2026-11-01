package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFavoritesStore(t *testing.T) {
	dir := t.TempDir()
	st := &Store{Path: filepath.Join(dir, "config.json"), Cfg: Config{}}

	if err := st.SetFavorite(Favorite{Kind: "modpack", ID: "abc", Name: "Pack"}, true); err != nil {
		t.Fatal(err)
	}
	if !st.Snapshot().IsFavorite("modpack", "abc") {
		t.Fatal("favorite should be stored")
	}
	if st.Snapshot().IsFavorite("resourcepack", "abc") {
		t.Fatal("different kinds must not match")
	}

	if err := st.SetFavorite(Favorite{Kind: "modpack", ID: "abc", Name: "Pack Renamed"}, true); err != nil {
		t.Fatal(err)
	}
	favs := st.Snapshot().FavoritesOf("modpack")
	if len(favs) != 1 || favs[0].Name != "Pack Renamed" {
		t.Fatalf("favorites = %#v", favs)
	}

	if err := st.SetFavorite(Favorite{Kind: "shader", ID: "xyz", Name: "Shader"}, true); err != nil {
		t.Fatal(err)
	}
	if len(st.Snapshot().FavoritesOf("modpack")) != 1 || len(st.Snapshot().FavoritesOf("shader")) != 1 {
		t.Fatalf("kind filtering failed")
	}

	if err := st.SetFavorite(Favorite{Kind: "modpack", ID: "abc"}, false); err != nil {
		t.Fatal(err)
	}
	if st.Snapshot().IsFavorite("modpack", "abc") {
		t.Fatal("favorite should be removed")
	}

	if err := st.SetFavorite(Favorite{Kind: "", ID: "x"}, true); err == nil {
		t.Fatal("expected invalid favorite error")
	}

	data, err := os.ReadFile(st.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("config should be persisted")
	}
}

func TestSnapshotCopiesFavorites(t *testing.T) {
	st := &Store{Path: filepath.Join(t.TempDir(), "config.json"), Cfg: Config{
		Favorites: []Favorite{{Kind: "modpack", ID: "a", Name: "A"}},
	}}
	snap := st.Snapshot()
	snap.Favorites[0].Name = "mutated"
	if st.Cfg.Favorites[0].Name != "A" {
		t.Fatal("snapshot must not alias favorites")
	}
}

func TestNormalizeLanguage(t *testing.T) {
	cases := map[string]string{"": "zh", "zh": "zh", "en": "en", "fr": "zh"}
	for in, want := range cases {
		cfg := Config{Language: in}
		normalize(&cfg)
		if cfg.Language != want {
			t.Fatalf("normalize(%q) = %q, want %q", in, cfg.Language, want)
		}
	}
}
