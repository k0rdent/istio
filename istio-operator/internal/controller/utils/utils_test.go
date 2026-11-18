package utils

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/k0rdent/istio/istio-operator/internal/hash"
)

const (
	ownNamespace     = "tenant-a"
	ownName          = "cluster-a"
	regionClusterKey = "k0rdent.mirantis.com/kcm-region-cluster"
)

type targetCluster struct {
	enabled   bool
	name      string
	namespace string
	cluster   string
	labels    map[string]any
}

func TestMustScopedCAPropagationServiceValuesYAML_BaseStructure(t *testing.T) {
	values := MustScopedCAPropagationServiceValuesYAML("CASecretData", ownNamespace, ownName)

	assertContainsAll(t, values,
		`(and (eq .Cluster.metadata.namespace "tenant-a") (eq .Cluster.metadata.name "cluster-a"))`,
		`(index .Cluster.metadata.labels "`+regionClusterKey+`")`,
		`enabled: {{ $eligible }}`,
		`data: |`,
		`{{ if $eligible }}`,
		`{{ copy "CASecretData" | nindent 14 }}`,
		`{{ end }}`,
	)
}

func TestMustScopedCAPropagationServiceValuesYAML_Rendering(t *testing.T) {
	values := MustScopedCAPropagationServiceValuesYAML("CASecretData", ownNamespace, ownName)

	assertRendering(t, values, []targetCluster{
		{name: "own cluster", namespace: ownNamespace, cluster: ownName, enabled: true},
		{name: "other cluster", namespace: "tenant-b", cluster: "cluster-b", enabled: false},
		{name: "same name other namespace", namespace: "tenant-b", cluster: ownName, enabled: false},
	})
	assertRegionRendering(t, values, true)
}

func TestMustSelfScopedPropagationServiceValuesYAML_BaseStructure(t *testing.T) {
	values := MustSelfScopedPropagationServiceValuesYAML("RemoteSecretData", ownNamespace, ownName)

	assertContainsAll(t, values,
		`{{- $eligible := (and (eq .Cluster.metadata.namespace "tenant-a") (eq .Cluster.metadata.name "cluster-a")) }}`,
		`enabled: {{ $eligible }}`,
		`{{ if $eligible }}`,
		`{{ copy "RemoteSecretData" | nindent 14 }}`,
	)
	if strings.Contains(values, regionClusterKey) {
		t.Fatalf("self-scoped values must not be enabled for region clusters, got:\n%s", values)
	}
}

func TestMustSelfScopedPropagationServiceValuesYAML_Rendering(t *testing.T) {
	values := MustSelfScopedPropagationServiceValuesYAML("RemoteSecretData", ownNamespace, ownName)

	assertRendering(t, values, []targetCluster{
		{name: "own cluster", namespace: ownNamespace, cluster: ownName, enabled: true},
		{name: "other cluster", namespace: "tenant-b", cluster: "cluster-b", enabled: false},
		{name: "same name other namespace", namespace: "tenant-b", cluster: ownName, enabled: false},
	})
	assertRegionRendering(t, values, false)
}

// Adler-32 checksums of different cluster identities may collide, so scoping must not rely on them.
func TestScopedPropagationValues_ChecksumCollision(t *testing.T) {
	const namespace, own, colliding = "default", "cluster-aca", "cluster-bab"
	if hash.Adler32sum(namespace+"-"+own) != hash.Adler32sum(namespace+"-"+colliding) {
		t.Fatalf("test precondition failed: expected %q and %q to have colliding checksums", own, colliding)
	}

	for name, values := range map[string]string{
		"self-scoped": MustSelfScopedPropagationServiceValuesYAML("Data", namespace, own),
		"CA-scoped":   MustScopedCAPropagationServiceValuesYAML("Data", namespace, own),
	} {
		t.Run(name, func(t *testing.T) {
			assertRendering(t, values, []targetCluster{
				{name: "own cluster", namespace: namespace, cluster: own, enabled: true},
				{name: "colliding cluster", namespace: namespace, cluster: colliding, enabled: false},
			})
		})
	}
}

func assertRegionRendering(t *testing.T, values string, enabled bool) {
	t.Helper()
	region := targetCluster{
		name: "region cluster", namespace: "tenant-b", cluster: "region",
		labels: map[string]any{regionClusterKey: "true"},
	}
	if got := renderEnabled(t, values, region); got != enabled {
		t.Errorf("%s: expected enabled=%v", region.name, enabled)
	}
}

func assertRendering(t *testing.T, values string, expected []targetCluster) {
	t.Helper()
	for _, target := range expected {
		if got := renderEnabled(t, values, target); got != target.enabled {
			t.Errorf("%s: expected enabled=%v, got %v", target.name, target.enabled, got)
		}
	}
}

// renderEnabled renders the values for the target cluster and reports whether propagation
// is enabled. It fails the test if the enabled flag and the presence of data disagree.
func renderEnabled(t *testing.T, values string, target targetCluster) bool {
	t.Helper()
	funcs := sprig.TxtFuncMap()
	funcs["copy"] = func(string) string { return "SECRET" }
	tmpl := template.Must(template.New("values").Funcs(funcs).Parse(values))

	var out bytes.Buffer
	if err := tmpl.Execute(&out, map[string]any{
		"Cluster": map[string]any{
			"metadata": map[string]any{
				"namespace": target.namespace,
				"name":      target.cluster,
				"labels":    target.labels,
			},
		},
	}); err != nil {
		t.Fatalf("%s: failed to render values: %v", target.name, err)
	}

	rendered := out.String()
	enabled := strings.Contains(rendered, "enabled: true")
	if hasData := strings.Contains(rendered, "SECRET"); hasData != enabled {
		t.Fatalf("%s: enabled=%v but data present=%v:\n%s", target.name, enabled, hasData, rendered)
	}
	return enabled
}

func assertContainsAll(t *testing.T, values string, checks ...string) {
	t.Helper()

	for _, check := range checks {
		if !strings.Contains(values, check) {
			t.Fatalf("expected values to contain %q, got:\n%s", check, values)
		}
	}
}
