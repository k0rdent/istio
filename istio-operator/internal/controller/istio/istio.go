package istio

import "fmt"

const (
	IstioRoleLabel = "k0rdent.mirantis.com/istio-role"
)

// `child` is deprecated but still supported
var IstioRoleLabelExpectedValues = []string{"member", "child"}

// By default "istio-system"
var IstioSystemNamespace string

// By default "k0rdent-istio"
var IstioReleaseName string

// By default ""; when set, service templates are referenced as
// <release-name>-<template>-<suffix>.
var IstioTemplateVersionSuffix string

func ServiceTemplateName(template string) string {
	if IstioTemplateVersionSuffix == "" {
		return fmt.Sprintf("%s-%s", IstioReleaseName, template)
	}

	return fmt.Sprintf("%s-%s-%s", IstioReleaseName, template, IstioTemplateVersionSuffix)
}
