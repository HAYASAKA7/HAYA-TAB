package platform

import "testing"

func TestCapabilitiesFor(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		width  int
		want   Capabilities
	}{
		{
			name:   "desktop",
			target: TargetDesktop,
			width:  1440,
			want: Capabilities{
				Target:             TargetDesktop,
				FormFactor:         FormFactorDesktop,
				LoopbackContent:    true,
				FolderWatcher:      true,
				CustomStoragePaths: true,
				Plugins:            true,
				WebMIDI:            true,
				SelfUpdate:         true,
			},
		},
		{
			name:   "iPhone",
			target: TargetIOS,
			width:  390,
			want: Capabilities{
				Target:             TargetIOS,
				FormFactor:         FormFactorPhone,
				NativeTopLevelTabs: true,
				InProcessContent:   true,
				NativeFileImport:   true,
				SafeAreaInsets:     true,
			},
		},
		{
			name:   "iPad",
			target: TargetIOS,
			width:  1024,
			want: Capabilities{
				Target:             TargetIOS,
				FormFactor:         FormFactorTablet,
				NativeTopLevelTabs: true,
				InProcessContent:   true,
				NativeFileImport:   true,
				SafeAreaInsets:     true,
			},
		},
		{
			name:   "Android phone",
			target: TargetAndroid,
			width:  412,
			want: Capabilities{
				Target:           TargetAndroid,
				FormFactor:       FormFactorPhone,
				WebTopLevelTabs:  true,
				InProcessContent: true,
				NativeFileImport: true,
				SafeAreaInsets:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CapabilitiesFor(tt.target, tt.width); got != tt.want {
				t.Fatalf("CapabilitiesFor(%q, %d) = %#v, want %#v", tt.target, tt.width, got, tt.want)
			}
		})
	}
}

func TestCapabilitiesForRejectsUnknownTarget(t *testing.T) {
	if got := CapabilitiesFor("webos", 390); got.Target != TargetDesktop {
		t.Fatalf("unknown targets must fail closed to desktop, got %#v", got)
	}
}
