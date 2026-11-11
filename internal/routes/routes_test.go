package routes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRoutesHostnameRules(t *testing.T) {
	for routes, wantErr := range map[string]string{
		`{"name":"a","upstream":"http://x:1"},{"name":"b","upstream":"http://x:2","public":true}`: "",
		`{"name":"A_b","upstream":"http://x:1"}`:                                                  "DNS label",
		`{"name":"a","upstream":"http://x:1"},{"name":"a","upstream":"http://x:2"}`:               "used twice",
		`{"name":"a","path":"/a","upstream":"http://x:1"}`:                                        "loopback port",
	} {
		f := filepath.Join(t.TempDir(), "routes.json")
		os.WriteFile(f, []byte(`{"idle_minutes":1,"routes":[`+routes+`]}`), 0o644)
		_, err := Load(f)
		if wantErr == "" && err != nil || wantErr != "" && (err == nil || !strings.Contains(err.Error(), wantErr)) {
			t.Errorf("%s: got %v, want %q", routes, err, wantErr)
		}
	}
}

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
		_, err := Load(f)
		if wantErr == "" && err != nil || wantErr != "" && (err == nil || !strings.Contains(err.Error(), wantErr)) {
			t.Errorf("%s: got %v, want %q", route, err, wantErr)
		}
	}
	f := filepath.Join(t.TempDir(), "routes.json")
	os.WriteFile(f, []byte(`{"idle_minutes":1,"routes":[
		{"name":"a","port":8444,"path":"/a","upstream":"http://x:1"},
		{"name":"b","port":8445,"path":"/a","upstream":"http://x:2"}]}`), 0o644)
	if _, err := Load(f); err == nil || !strings.Contains(err.Error(), "used twice") {
		t.Errorf("duplicate path: got %v", err)
	}
}
