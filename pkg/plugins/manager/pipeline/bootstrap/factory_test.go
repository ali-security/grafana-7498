package bootstrap

import (
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/grafana/grafana-app-sdk/app"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/log"
	"github.com/grafana/grafana/pkg/plugins/manager/pluginfakes"
)

const testManifestJSON = `{
  "apiVersion": "apps.grafana.app/v1alpha2",
  "kind": "AppManifest",
  "metadata": {
    "name": "test-app"
  },
  "spec": {
    "appName": "test-app",
    "group": "testapp.ext.grafana.com",
    "versions": [
      {
        "name": "v1",
        "kinds": [
          {
            "kind": "Thing",
            "plural": "things",
            "scope": "Namespaced"
          }
        ]
      }
    ]
  }
}`

func TestSetAppSDKManifests(t *testing.T) {
	t.Run("no manifest paths means no-op", func(t *testing.T) {
		p := &plugins.Plugin{
			JSONData: plugins.JSONData{},
		}
		err := setAppSDKManifests(p)
		require.NoError(t, err)
		require.Nil(t, p.AppSDKManifests)
	})

	t.Run("reads and parses manifest file", func(t *testing.T) {
		fakeFS := pluginfakes.NewFakePluginFS("/test")
		fakeFS.OpenFunc = func(name string) (fs.File, error) {
			if name == "my-app-manifest.json" {
				return newFakeFile(testManifestJSON), nil
			}
			return nil, fs.ErrNotExist
		}

		p := &plugins.Plugin{
			JSONData: plugins.JSONData{
				AppSDKManifest: []string{"my-app-manifest.json"},
			},
			FS: fakeFS,
		}
		p.SetLogger(log.NewTestLogger())

		err := setAppSDKManifests(p)
		require.NoError(t, err)
		require.Len(t, p.AppSDKManifests, 1)

		m := p.AppSDKManifests[0]
		require.Equal(t, app.ManifestLocationEmbedded, m.Location.Type)
		require.NotNil(t, m.ManifestData)
		require.Equal(t, "test-app", m.ManifestData.AppName)
		require.Equal(t, "testapp.ext.grafana.com", m.ManifestData.Group)
		require.Len(t, m.ManifestData.Versions, 1)
		require.Equal(t, "v1", m.ManifestData.Versions[0].Name)
	})

	t.Run("skips missing manifest file with warning", func(t *testing.T) {
		fakeFS := pluginfakes.NewFakePluginFS("/test")
		fakeFS.OpenFunc = func(name string) (fs.File, error) {
			return nil, fs.ErrNotExist
		}

		p := &plugins.Plugin{
			JSONData: plugins.JSONData{
				AppSDKManifest: []string{"missing-manifest.json"},
			},
			FS: fakeFS,
		}
		logger := log.NewTestLogger()
		p.SetLogger(logger)

		err := setAppSDKManifests(p)
		require.NoError(t, err)
		require.Empty(t, p.AppSDKManifests)
		require.NotZero(t, logger.WarnLogs.Calls)
	})

	t.Run("returns error for malformed JSON", func(t *testing.T) {
		fakeFS := pluginfakes.NewFakePluginFS("/test")
		fakeFS.OpenFunc = func(name string) (fs.File, error) {
			return newFakeFile("not valid json"), nil
		}

		p := &plugins.Plugin{
			JSONData: plugins.JSONData{
				AppSDKManifest: []string{"bad-manifest.json"},
			},
			FS: fakeFS,
		}
		p.SetLogger(log.NewTestLogger())

		err := setAppSDKManifests(p)
		require.Error(t, err)
		require.Contains(t, err.Error(), "bad-manifest.json")
	})

	t.Run("reads multiple manifest files", func(t *testing.T) {
		fakeFS := pluginfakes.NewFakePluginFS("/test")
		fakeFS.OpenFunc = func(name string) (fs.File, error) {
			return newFakeFile(testManifestJSON), nil
		}

		p := &plugins.Plugin{
			JSONData: plugins.JSONData{
				AppSDKManifest: []string{"manifest-a.json", "manifest-b.json"},
			},
			FS: fakeFS,
		}
		p.SetLogger(log.NewTestLogger())

		err := setAppSDKManifests(p)
		require.NoError(t, err)
		require.Len(t, p.AppSDKManifests, 2)
	})
}

// fakeFile implements fs.File for testing with in-memory content.
type fakeFile struct {
	io.Reader
}

func newFakeFile(content string) *fakeFile {
	return &fakeFile{Reader: strings.NewReader(content)}
}

func (f *fakeFile) Stat() (fs.FileInfo, error) { return nil, nil }
func (f *fakeFile) Close() error                { return nil }
