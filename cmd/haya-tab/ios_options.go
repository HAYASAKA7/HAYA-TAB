package main

import "github.com/wailsapp/wails/v3/pkg/application"

func iosNativeTabs(locale string) []application.NativeTabItem {
	titles := map[string][4]string{
		"en":    {"Library", "Offline", "Search", "Settings"},
		"ja":    {"ライブラリ", "オフライン", "検索", "設定"},
		"zh-CN": {"曲库", "离线", "搜索", "设置"},
		"zh-TW": {"曲庫", "離線", "搜尋", "設置"},
	}
	labels, ok := titles[locale]
	if !ok {
		labels = titles["en"]
	}

	return []application.NativeTabItem{
		{Title: labels[0], SystemImage: application.NativeTabIcon("books.vertical")},
		{Title: labels[1], SystemImage: application.NativeTabIcon("arrow.down.circle")},
		{Title: labels[2], SystemImage: application.NativeTabIconMagnify},
		{Title: labels[3], SystemImage: application.NativeTabIconGear},
	}
}

func applyIOSOptions(opts *application.Options, locale string) {
	opts.DisableDefaultSignalHandler = true
	opts.IOS.EnableNativeTabs = true
	opts.IOS.NativeTabsItems = iosNativeTabs(locale)
	opts.IOS.EnableBackForwardNavigationGestures = true
}
