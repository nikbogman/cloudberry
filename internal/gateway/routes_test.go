package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRoutesPathRules(t *testing.T) {
	for route, wantErr := range map[string]string{
		`{"name":"a","port":8444,"path":"/a","upstream":"http://x:1"}`:               "",
		`{"name":"a","port":8444,"path":"/","upstream":"http://x:1"}`:                "path must",
		`{"name":"a","port":8444,"path":"a","upstream":"http://x:1"}`:                "path must",
		`{"name":"a","port":8444,"path":"/a/","upstream":"http://x:1"}`:              "path must",
		`{"name":"a","port":8443,"path":"/a","upstream":"http://x:1","public":true}`: "can't be public",
	} {
		f := filepath.Join(t.TempDir(), "routes.json")
		os.WriteFile(f, []byte(`{"idle_minutes":1,"routes":[`+route+`]}`), 0o644)
		_, err := LoadRoutes(f)
		if wantErr == "" && err != nil || wantErr != "" && (err == nil || !strings.Contains(err.Error(), wantErr)) {
			t.Errorf("%s: got %v, want %q", route, err, wantErr)
		}
	}
	f := filepath.Join(t.TempDir(), "routes.json")
	os.WriteFile(f, []byte(`{"idle_minutes":1,"routes":[
		{"name":"a","port":8444,"path":"/a","upstream":"http://x:1"},
		{"name":"b","port":8445,"path":"/a","upstream":"http://x:2"}]}`), 0o644)
	if _, err := LoadRoutes(f); err == nil || !strings.Contains(err.Error(), "used twice") {
		t.Errorf("duplicate path: got %v", err)
	}
}
