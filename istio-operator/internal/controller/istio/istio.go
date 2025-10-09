package istio

const (
	IstioRoleLabel = "k0rdent.mirantis.com/istio-role"
)

// `child` is deprecated but still supported
var IstioRoleLabelExpectedValues = []string{"member", "child"}

// By default "istio-system"
var IstioSystemNamespace string

// By default "k0rdent-istio"
var IstioReleaseName string
