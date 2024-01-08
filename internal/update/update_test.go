package update

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/stretchr/testify/require"
)

func TestLatestVersion(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name       string
		resp       *ReleaseInfo
		statusCode int
	}{
		{
			name:       "valid response",
			statusCode: 200,
			resp: &ReleaseInfo{
				Version: "v0.1.0",
			},
		},
		{
			name:       "non valid response",
			statusCode: 400,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.resp)
			}))
			defer ts.Close()

			info, err := latestVersion(context.Background(), ts.URL)

			success := tt.statusCode >= 200 && tt.statusCode < 300
			if !success {
				c.Assert(err, qt.Not(qt.IsNil))
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(info, qt.DeepEquals, tt.resp)
			}
		})
	}
}

func TestCheckVersion(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name          string
		buildVersion  string
		latestVersion string
		update        bool
		lastChecked   time.Time
	}{
		{
			name:          "self-compiled",
			buildVersion:  "",
			latestVersion: "v0.2.0",
			update:        false,
		},
		{
			name:          "new version",
			buildVersion:  "v0.1.0",
			latestVersion: "v0.2.0",
			update:        true,
		},
		{
			name:          "same version",
			buildVersion:  "v0.2.0",
			latestVersion: "v0.2.0",
			update:        false,
		},
		{
			name:          "higher version",
			buildVersion:  "v0.3.0",
			latestVersion: "v0.2.0",
			update:        false,
		},
		{
			name:          "new version, but we already checked in the past 24 hours",
			buildVersion:  "v0.1.0",
			latestVersion: "v0.2.0",
			update:        false,
			lastChecked:   time.Now().Add(-time.Hour),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "state.yml")

			if !tt.lastChecked.IsZero() {
				err := setStateEntry(path, tt.lastChecked, ReleaseInfo{Version: tt.latestVersion})
				c.Assert(err, qt.IsNil)
			}

			updateInfo, err := checkVersion(
				context.Background(),
				tt.buildVersion,
				path,
				func(ctx context.Context, addr string) (*ReleaseInfo, error) {
					return &ReleaseInfo{Version: tt.latestVersion}, nil
				},
			)

			c.Assert(err, qt.IsNil)
			c.Assert(updateInfo.Update, qt.Equals, tt.update, qt.Commentf("reason: %s", updateInfo.Reason))

			if !tt.update {
				t.Logf("reason: %s", updateInfo.Reason)
			}
		})
	}
}

func Test_setStateEntry(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) string
		err   error
	}{
		{
			name: "create file",
			setup: func(t *testing.T) string {
				const name = "teststate.yml"
				t.Cleanup(func() { _ = os.Remove(name) })

				return name
			},
		},
		{
			name: "create directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()

				return filepath.Join(dir, "state.yml")
			},
		},
		{
			name: "create nested directories",
			setup: func(t *testing.T) string {
				dir := t.TempDir()

				return filepath.Join(dir, "dir1", "dir2", "state.yml")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)

			err := setStateEntry(path, time.Unix(1, 0), ReleaseInfo{})
			require.ErrorIs(t, err, tt.err)
			if err != nil {
				return
			}

			stat, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, fs.FileMode(0o644), stat.Mode().Perm())
		})
	}
}
