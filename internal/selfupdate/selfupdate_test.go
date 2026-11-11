package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// release serves a fake GitHub: latest is tag, with asset "app" = body and
// SHA256SUMS listing sumOf (normally body's hash).
func release(t *testing.T, tag, body, sumOf string) string {
	h := sha256.Sum256([]byte(sumOf))
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/releases/tag/"+tag, http.StatusFound)
	})
	mux.HandleFunc("/releases/download/"+tag+"/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(hex.EncodeToString(h[:]) + "  app\n"))
	})
	mux.HandleFunc("/releases/download/"+tag+"/app", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func setup(t *testing.T, url, version string) (*Updater, string) {
	bin := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(bin, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &Updater{BaseURL: url, Asset: "app", BinPath: bin, Version: version}, bin
}

func read(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return "<missing>"
	}
	return string(b)
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("new release is swapped in, old kept as .prev", func(t *testing.T) {
		u, bin := setup(t, release(t, "v1.1.0", "new", "new"), "v1.0.0")
		tag, updated, err := u.Update(ctx)
		if err != nil || !updated || tag != "v1.1.0" || read(t, bin) != "new" || read(t, bin+".prev") != "old" {
			t.Fatalf("tag=%s updated=%v err=%v bin=%s prev=%s", tag, updated, err, read(t, bin), read(t, bin+".prev"))
		}
	})
	t.Run("already on latest", func(t *testing.T) {
		u, bin := setup(t, release(t, "v1.0.0", "new", "new"), "v1.0.0")
		if _, updated, err := u.Update(ctx); err != nil || updated || read(t, bin) != "old" {
			t.Fatalf("updated=%v err=%v", updated, err)
		}
	})
	t.Run("local build is left alone", func(t *testing.T) {
		u, bin := setup(t, release(t, "v1.1.0", "new", "new"), "v1.0.0-3-gabc123-dirty")
		if _, updated, err := u.Update(ctx); err != ErrLocalBuild || updated || read(t, bin) != "old" {
			t.Fatalf("updated=%v err=%v", updated, err)
		}
	})
	t.Run("bad checksum leaves the old binary", func(t *testing.T) {
		u, bin := setup(t, release(t, "v1.1.0", "tampered", "new"), "v1.0.0")
		if _, updated, err := u.Update(ctx); err == nil || updated || read(t, bin) != "old" || read(t, bin+".prev") != "<missing>" {
			t.Fatalf("updated=%v err=%v", updated, err)
		}
	})
	t.Run("failed download leaves the old binary", func(t *testing.T) {
		u, bin := setup(t, release(t, "v1.1.0", "new", "new"), "v1.0.0")
		u.Asset = "missing"
		if _, updated, err := u.Update(ctx); err == nil || updated || read(t, bin) != "old" {
			t.Fatalf("updated=%v err=%v", updated, err)
		}
	})
	t.Run("a rolled-back release isn't installed again", func(t *testing.T) {
		u, bin := setup(t, release(t, "v1.1.0", "new", "new"), "v1.0.0")
		os.WriteFile(bin+".bad", []byte("new"), 0o755)
		if _, updated, err := u.Update(ctx); err == nil || !strings.Contains(err.Error(), "rolled back") || updated || read(t, bin) != "old" {
			t.Fatalf("updated=%v err=%v", updated, err)
		}
	})
}

func TestNeverDowngrades(t *testing.T) {
	// Between a tag push and its release, /releases/latest still names the
	// previous release: a build of the new tag must not "update" back to it.
	u, bin := setup(t, release(t, "v0.1.1", "old-release", "old-release"), "v0.2.0")
	if _, updated, err := u.Update(context.Background()); err != nil || updated || read(t, bin) != "old" {
		t.Fatalf("updated=%v err=%v bin=%s", updated, err, read(t, bin))
	}
	for a, b := range map[string]string{"v0.10.0": "v0.9.9", "v1.0.0": "v0.99.99", "v0.2.1": "v0.2.0"} {
		if !newer(a, b) || newer(b, a) || newer(a, a) {
			t.Errorf("newer(%s, %s) wrong", a, b)
		}
	}
}

func TestFollowsRepoRename(t *testing.T) {
	// A renamed GitHub repo answers the old /releases/latest with a redirect
	// to the new repo's /releases/latest, which redirects to the tag.
	newURL := release(t, "v1.1.0", "new", "new")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, newURL+r.URL.Path, http.StatusMovedPermanently)
	})
	old := httptest.NewServer(mux)
	t.Cleanup(old.Close)

	u, bin := setup(t, old.URL, "v1.0.0")
	if tag, updated, err := u.Update(context.Background()); err != nil || !updated || tag != "v1.1.0" || read(t, bin) != "new" {
		t.Fatalf("tag=%s updated=%v err=%v", tag, updated, err)
	}
}
