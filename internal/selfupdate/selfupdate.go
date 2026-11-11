// Package selfupdate swaps a running binary for the asset of the same name in
// the latest GitHub release. The swap is a rename, so a crash mid-update
// leaves the old binary in place; systemd restarts the service on the new one,
// and its rollback unit restores <bin>.prev if that keeps failing.
package selfupdate

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Release tags look like v1.2.3. Anything else (a `git describe` from a
// local build, "dev") is a binary someone deployed by hand: it's left alone.
var releaseTag = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// Updater checks Repo's latest release for Asset.
type Updater struct {
	BaseURL string // https://github.com/<owner>/<repo>
	Asset   string // e.g. gateway-linux-armv6
	BinPath string // the running binary, e.g. /usr/local/bin/gateway
	Version string // the running binary's version, from -ldflags
	Client  *http.Client
}

// ErrLocalBuild means Version isn't a release tag, so nothing is updated.
var ErrLocalBuild = errors.New("not a release build; not updating")

// Update installs the latest release if it's not the running one. updated
// is true when a new binary is in place and the caller should exit so systemd
// restarts it. A release whose hash matches <bin>.bad, which the rollback
// unit leaves behind, is skipped.
func (u *Updater) Update(ctx context.Context) (tag string, updated bool, err error) {
	if !releaseTag.MatchString(u.Version) {
		return "", false, ErrLocalBuild
	}
	client := u.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	tag, err = u.latestTag(ctx, client)
	if err != nil || !newer(tag, u.Version) {
		return tag, false, err
	}
	sums, err := get(ctx, client, u.BaseURL+"/releases/download/"+tag+"/SHA256SUMS")
	if err != nil {
		return tag, false, err
	}
	want, err := sumFor(sums, u.Asset)
	if err != nil {
		return tag, false, err
	}
	if bad, err := fileSum(u.BinPath + ".bad"); err == nil && bad == want {
		return tag, false, fmt.Errorf("%s %s was rolled back before; waiting for a newer release", u.Asset, tag)
	}
	bin, err := get(ctx, client, u.BaseURL+"/releases/download/"+tag+"/"+u.Asset)
	if err != nil {
		return tag, false, err
	}
	if got := sum(bin); got != want {
		return tag, false, fmt.Errorf("%s %s: sha256 %s, SHA256SUMS says %s", u.Asset, tag, got, want)
	}
	if err := os.WriteFile(u.BinPath+".new", bin, 0o755); err != nil {
		return tag, false, err
	}
	// Two renames, never an in-place write: a running binary can't be
	// overwritten (ETXTBSY), and a crash between them leaves the old one.
	if err := os.Rename(u.BinPath, u.BinPath+".prev"); err != nil {
		return tag, false, err
	}
	if err := os.Rename(u.BinPath+".new", u.BinPath); err != nil {
		os.Rename(u.BinPath+".prev", u.BinPath)
		return tag, false, err
	}
	return tag, true, nil
}

// latestTag reads the tag from github.com's /releases/latest redirect: not
// the REST API, whose 60 requests an hour per IP two devices would share.
func (u *Updater) latestTag(ctx context.Context, client *http.Client) (string, error) {
	c := *client
	// Follow redirects (a renamed repo redirects to its new name) up to the
	// one that names the tag, and read that one's Location.
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if strings.Contains(req.URL.Path, "/releases/tag/") || len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u.BaseURL+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	tag := path.Base(loc)
	if resp.StatusCode/100 != 3 || !strings.Contains(loc, "/releases/tag/") || !releaseTag.MatchString(tag) {
		return "", fmt.Errorf("latest release: status %d, location %q", resp.StatusCode, loc)
	}
	return tag, nil
}

// newer reports whether release tag a is a later version than b. Both match
// releaseTag. Only ever forward: between a tag push and its release, latest
// still names the previous one.
func newer(a, b string) bool {
	pa, pb := strings.Split(a[1:], "."), strings.Split(b[1:], ".")
	for i := range pa {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x > y
		}
	}
	return false
}

func get(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// sumFor finds asset's hash in sha256sum output ("<hex>  <name>").
func sumFor(sums []byte, asset string) (string, error) {
	s := bufio.NewScanner(bytes.NewReader(sums))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == asset {
			return f[0], nil
		}
	}
	return "", fmt.Errorf("%s not in SHA256SUMS", asset)
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func fileSum(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return sum(b), nil
}
