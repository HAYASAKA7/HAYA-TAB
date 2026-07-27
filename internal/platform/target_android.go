//go:build android

package platform

func CurrentTarget() Target {
	return TargetAndroid
}
