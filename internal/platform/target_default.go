//go:build !ios && !android

package platform

func CurrentTarget() Target {
	return TargetDesktop
}
