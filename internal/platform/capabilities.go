package platform

type Target string
type FormFactor string

const (
	TargetDesktop Target = "desktop"
	TargetIOS     Target = "ios"
	TargetAndroid Target = "android"

	FormFactorPhone   FormFactor = "phone"
	FormFactorTablet  FormFactor = "tablet"
	FormFactorDesktop FormFactor = "desktop"
)

type Capabilities struct {
	Target             Target     `json:"target"`
	FormFactor         FormFactor `json:"formFactor"`
	NativeTopLevelTabs bool       `json:"nativeTopLevelTabs"`
	WebTopLevelTabs    bool       `json:"webTopLevelTabs"`
	InProcessContent   bool       `json:"inProcessContent"`
	LoopbackContent    bool       `json:"loopbackContent"`
	NativeFileImport   bool       `json:"nativeFileImport"`
	SafeAreaInsets     bool       `json:"safeAreaInsets"`
	FolderWatcher      bool       `json:"folderWatcher"`
	CustomStoragePaths bool       `json:"customStoragePaths"`
	Plugins            bool       `json:"plugins"`
	WebMIDI            bool       `json:"webMIDI"`
	SelfUpdate         bool       `json:"selfUpdate"`
}

func CapabilitiesFor(target Target, viewportWidth int) Capabilities {
	if target != TargetIOS && target != TargetAndroid {
		return Capabilities{
			Target:             TargetDesktop,
			FormFactor:         FormFactorDesktop,
			LoopbackContent:    true,
			FolderWatcher:      true,
			CustomStoragePaths: true,
			Plugins:            true,
			WebMIDI:            true,
			SelfUpdate:         true,
		}
	}

	formFactor := FormFactorPhone
	if viewportWidth >= 768 {
		formFactor = FormFactorTablet
	}
	result := Capabilities{
		Target:           target,
		FormFactor:       formFactor,
		InProcessContent: true,
		NativeFileImport: true,
		SafeAreaInsets:   true,
	}
	result.NativeTopLevelTabs = target == TargetIOS
	result.WebTopLevelTabs = target == TargetAndroid
	return result
}
