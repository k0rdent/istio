package utils

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/k0rdent/istio/istio-operator/internal/hash"
)

func TestMustScopedCAPropagationServiceValuesYAML_BaseStructure(t *testing.T) {
	values := MustScopedCAPropagationServiceValuesYAML("CASecretData", "istio-ca-secret-2484144794")

	assertContainsAll(t, values,
		`{{- $eligible := or (eq (adler32sum (printf "%s-%s" .Cluster.metadata.namespace .Cluster.metadata.name)) (last (splitList "-" "istio-ca-secret-2484144794"))) (and .Cluster.metadata.labels (eq (index .Cluster.metadata.labels "k0rdent.mirantis.com/kcm-region-cluster") "true")) }}`,
		`enabled: {{ $eligible }}`,
		`data: |`,
		`{{ if $eligible }}`,
		`{{ copy "CASecretData" | nindent 14 }}`,
		`{{ end }}`,
	)
}

func TestMustScopedCAPropagationServiceValuesYAML_DifferentInputs(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		secretName string
	}{
		{name: "primary hash", identifier: "CASecretData", secretName: "istio-ca-secret-2484144794"},
		{name: "secondary hash", identifier: "SecretPayload", secretName: "istio-ca-secret-1946159437"},
		{name: "short hash", identifier: "Data", secretName: "x-42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := MustScopedCAPropagationServiceValuesYAML(tt.identifier, tt.secretName)
			assertContainsAll(t, values,
				`{{- $eligible := or (eq (adler32sum (printf "%s-%s" .Cluster.metadata.namespace .Cluster.metadata.name)) (last (splitList "-" "`+tt.secretName+`")))`,
				`enabled: {{ $eligible }}`,
				`{{ if $eligible }}`,
				`index .Cluster.metadata.labels "k0rdent.mirantis.com/kcm-region-cluster"`,
				`{{ copy "`+tt.identifier+`" | nindent 14 }}`,
			)
		})
	}
}

func TestMustScopedCAPropagationServiceValuesYAML_UsesBlockConditionals(t *testing.T) {
	values := MustScopedCAPropagationServiceValuesYAML("CASecretData", "istio-ca-secret-2484144794")
	for _, expected := range []string{"{{ if", "{{ end }}"} {
		if !strings.Contains(values, expected) {
			t.Fatalf("expected values to contain %q, got:\n%s", expected, values)
		}
	}
}

func assertContainsAll(t *testing.T, values string, checks ...string) {
	t.Helper()

	for _, check := range checks {
		if !strings.Contains(values, check) {
			t.Fatalf("expected values to contain %q, got:\n%s", check, values)
		}
	}
}

func TestMustSelfScopedPropagationServiceValuesYAML_BaseStructure(t *testing.T) {
	values := MustSelfScopedPropagationServiceValuesYAML("RemoteSecretData", "istio-remote-secret-2484144794")

	assertContainsAll(t, values,
		`{{- $eligible := eq (adler32sum (printf "%s-%s" .Cluster.metadata.namespace .Cluster.metadata.name)) (last (splitList "-" "istio-remote-secret-2484144794")) }}`,
		`enabled: {{ $eligible }}`,
		`{{ if $eligible }}`,
		`{{ copy "RemoteSecretData" | nindent 14 }}`,
	)
	if strings.Contains(values, "kcm-region-cluster") {
		t.Fatalf("self-scoped values must not be enabled for region clusters, got:\n%s", values)
	}
}

func TestMustSelfScopedPropagationServiceValuesYAML_RendersOnlyForOwnCluster(t *testing.T) {
	const ownNamespace, ownName = "tenant-a", "cluster-a"
	secretName := hash.WithPrefix("istio-remote-secret", ownNamespace+"-"+ownName, hash.AdlerHash)
	values := MustSelfScopedPropagationServiceValuesYAML("RemoteSecretData", secretName)

	tests := []struct {
		name      string
		namespace string
		cluster   string
		labels    map[string]any
		enabled   bool
	}{
		{name: "own cluster", namespace: ownNamespace, cluster: ownName, enabled: true},
		{name: "other tenant cluster", namespace: "tenant-b", cluster: "cluster-b", enabled: false},
		{
			name: "region cluster", namespace: "tenant-b", cluster: "region",
			labels: map[string]any{"k0rdent.mirantis.com/kcm-region-cluster": "true"}, enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcs := sprig.TxtFuncMap()
			funcs["copy"] = func(string) string { return "SECRET" }
			tmpl := template.Must(template.New("values").Funcs(funcs).Parse(values))

			var out bytes.Buffer
			if err := tmpl.Execute(&out, map[string]any{
				"Cluster": map[string]any{
					"metadata": map[string]any{
						"namespace": tt.namespace,
						"name":      tt.cluster,
						"labels":    tt.labels,
					},
				},
			}); err != nil {
				t.Fatalf("failed to render values: %v", err)
			}

			rendered := out.String()
			if got := strings.Contains(rendered, "enabled: true"); got != tt.enabled {
				t.Errorf("expected enabled=%v, got:\n%s", tt.enabled, rendered)
			}
			if got := strings.Contains(rendered, "SECRET"); got != tt.enabled {
				t.Errorf("expected secret data present=%v, got:\n%s", tt.enabled, rendered)
			}
		})
	}
}
