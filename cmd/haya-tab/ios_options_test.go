package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestApplyIOSOptionsConfiguresNativeTabs(t *testing.T) {
	opts := application.Options{}
	applyIOSOptions(&opts, "en")

	if !opts.DisableDefaultSignalHandler {
		t.Fatal("iOS must disable default signal handling")
	}
	if !opts.IOS.EnableNativeTabs {
		t.Fatal("iOS native tabs must be enabled")
	}
	if !opts.IOS.EnableBackForwardNavigationGestures {
		t.Fatal("iOS back/forward gestures must be enabled")
	}
	if got := len(opts.IOS.NativeTabsItems); got != 4 {
		t.Fatalf("native tabs = %d, want 4", got)
	}
}

func TestIOSNativeTabsUsesSupportedLocale(t *testing.T) {
	if got := iosNativeTabs("ja")[0].Title; got != "ライブラリ" {
		t.Fatalf("Japanese library title = %q", got)
	}
	if got := iosNativeTabs("unsupported")[0].Title; got != "Library" {
		t.Fatalf("fallback library title = %q", got)
	}
}

func TestIOSNativeTabsUsesSystemSFIcons(t *testing.T) {
	items := iosNativeTabs("en")
	want := []application.NativeTabIcon{
		application.NativeTabIcon("books.vertical"),
		application.NativeTabIcon("arrow.down.circle"),
		application.NativeTabIconMagnify,
		application.NativeTabIconGear,
	}

	for index, item := range items {
		if item.SystemImage != want[index] {
			t.Fatalf("tab %d icon = %q, want %q", index, item.SystemImage, want[index])
		}
	}
}
