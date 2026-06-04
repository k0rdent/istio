package env

import "os"

const (
	// `RELEASE_VERSION` is used to set the version of the release being deployed. It is used in the names of the resources created by the operator, to allow multiple versions of the operator to coexist in the same cluster.
	ReleaseVersionEnv = "RELEASE_VERSION"
	// `SELF_MANAGEMENT` is used to indicate whether the operator should manage itself. It is used to enable or disable self-management features in the operator.
	SelfManagementEnv = "SELF_MANAGEMENT"
	// `MANAGEMENT_CLUSTER_API_SERVER` is the externally accessible URL of the management cluster API server.
	// Required when running in-cluster so that Istiod on member clusters can reach the management cluster API server.
	ManagementClusterAPIServerEnv = "MANAGEMENT_CLUSTER_API_SERVER"
)

// These constants are used to represent boolean values as strings, since environment variables are always strings.
const (
	True  = "true"
	False = "false"
)

func GetEnvOrDefault(envVar, defaultValue string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetReleaseVersion() string {
	return GetEnvOrDefault(ReleaseVersionEnv, "")
}

func IsSelfManagementEnabled() bool {
	return GetEnvOrDefault(SelfManagementEnv, False) == True
}

func GetManagementClusterAPIServer() string {
	return GetEnvOrDefault(ManagementClusterAPIServerEnv, "")
}
