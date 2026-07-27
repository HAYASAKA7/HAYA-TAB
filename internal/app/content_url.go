package app

import (
	"fmt"
	"net/url"

	"haya-tab/internal/platform"
)

func contentURL(target platform.Target, port int, kind, id string) (string, error) {
	switch kind {
	case "file", "cover", "cloud-stream":
	default:
		return "", fmt.Errorf("unsupported content kind %q", kind)
	}
	if id == "" {
		return "", fmt.Errorf("content identifier must not be empty")
	}

	path := fmt.Sprintf("/api/%s/%s", kind, url.PathEscape(id))
	switch target {
	case platform.TargetIOS, platform.TargetAndroid:
		return path, nil
	case platform.TargetDesktop:
		if port <= 0 || port > 65535 {
			return "", fmt.Errorf("desktop file server is not running")
		}
		return fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil
	default:
		return "", fmt.Errorf("unsupported runtime target %q", target)
	}
}

func (a *App) GetTabContentURL(id string) (string, error) {
	return contentURL(platform.CurrentTarget(), a.GetFileServerPort(), "file", id)
}

func (a *App) GetCoverContentURL(id string) (string, error) {
	return contentURL(platform.CurrentTarget(), a.GetFileServerPort(), "cover", id)
}
