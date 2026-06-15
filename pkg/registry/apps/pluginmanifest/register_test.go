package pluginmanifest

import (
	"testing"

	"github.com/grafana/grafana-app-sdk/app"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/manager/pluginfakes"
)

func TestProvideAppInstallers(t *testing.T) {
	t.Run("returns no installers when no plugins have manifests", func(t *testing.T) {
		reg := pluginfakes.NewFakePluginRegistry()
		reg.Store["test-app"] = &plugins.Plugin{
			JSONData: plugins.JSONData{
				ID:   "test-app",
				Type: plugins.TypeApp,
			},
		}

		installers, err := ProvideAppInstallers(reg, nil, nil)
		require.NoError(t, err)
		require.Empty(t, installers)
	})

	t.Run("creates installer for plugin with manifest", func(t *testing.T) {
		manifest := app.NewEmbeddedManifest(app.ManifestData{
			AppName: "test-app",
			Group:   "testapp.ext.grafana.com",
			Versions: []app.ManifestVersion{
				{
					Name: "v1",
					Kinds: []app.ManifestVersionKind{
						{
							Kind:   "Thing",
							Plural: "things",
							Scope:  "Namespaced",
						},
					},
				},
			},
		})

		reg := pluginfakes.NewFakePluginRegistry()
		reg.Store["test-app"] = &plugins.Plugin{
			JSONData: plugins.JSONData{
				ID:   "test-app",
				Type: plugins.TypeApp,
			},
			AppSDKManifests: []app.Manifest{manifest},
		}

		installers, err := ProvideAppInstallers(reg, nil, nil)
		require.NoError(t, err)
		require.Len(t, installers, 1)
		require.Equal(t, []schema.GroupVersion{{Group: "testapp.ext.grafana.com", Version: "v1"}}, installers[0].GroupVersions())
	})

	t.Run("creates multiple installers for multiple manifests", func(t *testing.T) {
		m1 := app.NewEmbeddedManifest(app.ManifestData{
			AppName: "app-one",
			Group:   "appone.ext.grafana.com",
			Versions: []app.ManifestVersion{
				{Name: "v1", Kinds: []app.ManifestVersionKind{{Kind: "Foo", Plural: "foos", Scope: "Namespaced"}}},
			},
		})
		m2 := app.NewEmbeddedManifest(app.ManifestData{
			AppName: "app-two",
			Group:   "apptwo.ext.grafana.com",
			Versions: []app.ManifestVersion{
				{Name: "v1", Kinds: []app.ManifestVersionKind{{Kind: "Bar", Plural: "bars", Scope: "Namespaced"}}},
			},
		})

		reg := pluginfakes.NewFakePluginRegistry()
		reg.Store["multi-app"] = &plugins.Plugin{
			JSONData: plugins.JSONData{
				ID:   "multi-app",
				Type: plugins.TypeApp,
			},
			AppSDKManifests: []app.Manifest{m1, m2},
		}

		installers, err := ProvideAppInstallers(reg, nil, nil)
		require.NoError(t, err)
		require.Len(t, installers, 2)
	})

	t.Run("skips non-app plugins", func(t *testing.T) {
		manifest := app.NewEmbeddedManifest(app.ManifestData{
			AppName: "ds-app",
			Group:   "dsapp.ext.grafana.com",
			Versions: []app.ManifestVersion{
				{Name: "v1", Kinds: []app.ManifestVersionKind{{Kind: "X", Plural: "xs", Scope: "Namespaced"}}},
			},
		})

		reg := pluginfakes.NewFakePluginRegistry()
		reg.Store["test-datasource"] = &plugins.Plugin{
			JSONData: plugins.JSONData{
				ID:   "test-datasource",
				Type: plugins.TypeDataSource,
			},
			AppSDKManifests: []app.Manifest{manifest},
		}

		installers, err := ProvideAppInstallers(reg, nil, nil)
		require.NoError(t, err)
		require.Empty(t, installers)
	})

	t.Run("no admission plugin when manifest declares no admission", func(t *testing.T) {
		manifest := app.NewEmbeddedManifest(app.ManifestData{
			AppName: "test-app",
			Group:   "testapp.ext.grafana.com",
			Versions: []app.ManifestVersion{
				{Name: "v1", Kinds: []app.ManifestVersionKind{{Kind: "Thing", Plural: "things", Scope: "Namespaced"}}},
			},
		})

		reg := pluginfakes.NewFakePluginRegistry()
		reg.Store["test-app"] = &plugins.Plugin{
			JSONData:        plugins.JSONData{ID: "test-app", Type: plugins.TypeApp},
			AppSDKManifests: []app.Manifest{manifest},
		}

		installers, err := ProvideAppInstallers(reg, nil, nil)
		require.NoError(t, err)
		require.Len(t, installers, 1)
		require.Nil(t, installers[0].AdmissionPlugin())
	})

	t.Run("registers admission plugin when a kind declares validation", func(t *testing.T) {
		manifest := app.NewEmbeddedManifest(app.ManifestData{
			AppName: "test-app",
			Group:   "testapp.ext.grafana.com",
			Versions: []app.ManifestVersion{
				{Name: "v1", Kinds: []app.ManifestVersionKind{{
					Kind:   "Thing",
					Plural: "things",
					Scope:  "Namespaced",
					Admission: &app.AdmissionCapabilities{
						Validation: &app.ValidationCapability{
							Operations: []app.AdmissionOperation{app.AdmissionOperationCreate},
						},
					},
				}}},
			},
		})

		reg := pluginfakes.NewFakePluginRegistry()
		reg.Store["test-app"] = &plugins.Plugin{
			JSONData:        plugins.JSONData{ID: "test-app", Type: plugins.TypeApp},
			AppSDKManifests: []app.Manifest{manifest},
		}

		installers, err := ProvideAppInstallers(reg, nil, nil)
		require.NoError(t, err)
		require.Len(t, installers, 1)
		require.NotNil(t, installers[0].AdmissionPlugin())
	})
}
