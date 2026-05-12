package istio

import (
	"fmt"

	"github.com/k0rdent/istio/istio-operator/internal/labels"
)

// `child` is deprecated but still supported
var IstioRoleLabelExpectedValues = []string{
	labels.IstioRoleLabelMemberValue,
	labels.IstioRoleLabelChildValue,
}

// By default "istio-system"
var IstioSystemNamespace string

// By default "k0rdent-istio"
var IstioReleaseName string

// By default is empty string, but can be set by the `release-version` flag or the `RELEASE_VERSION` environment variable.
var ReleaseVersion string

func ServiceTemplateName(template string) string {
	if ReleaseVersion == "" {
		return fmt.Sprintf("%s-%s", IstioReleaseName, template)
	}
	return fmt.Sprintf("%s-%s-%s", IstioReleaseName, template, ReleaseVersion)
}
