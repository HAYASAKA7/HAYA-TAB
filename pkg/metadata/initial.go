package metadata

import (
	"strings"
	"unicode"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"github.com/mozillazg/go-pinyin"
)

// Gojūon row mapping: maps Kana characters to their row representative
var kanaToRow = map[rune]string{
	// Hiragana あ行
	'あ': "あ", 'い': "あ", 'う': "あ", 'え': "あ", 'お': "あ",
	// Hiragana か行
	'か': "か", 'き': "か", 'く': "か", 'け': "か", 'こ': "か",
	'が': "か", 'ぎ': "か", 'ぐ': "か", 'げ': "か", 'ご': "か",
	// Hiragana さ行
	'さ': "さ", 'し': "さ", 'す': "さ", 'せ': "さ", 'そ': "さ",
	'ざ': "さ", 'じ': "さ", 'ず': "さ", 'ぜ': "さ", 'ぞ': "さ",
	// Hiragana た行
	'た': "た", 'ち': "た", 'つ': "た", 'て': "た", 'と': "た",
	'だ': "た", 'ぢ': "た", 'づ': "た", 'で': "た", 'ど': "た",
	// Hiragana な行
	'な': "な", 'に': "な", 'ぬ': "な", 'ね': "な", 'の': "な",
	// Hiragana は行
	'は': "は", 'ひ': "は", 'ふ': "は", 'へ': "は", 'ほ': "は",
	'ば': "は", 'び': "は", 'ぶ': "は", 'べ': "は", 'ぼ': "は",
	'ぱ': "は", 'ぴ': "は", 'ぷ': "は", 'ぺ': "は", 'ぽ': "は",
	// Hiragana ま行
	'ま': "ま", 'み': "ま", 'む': "ま", 'め': "ま", 'も': "ま",
	// Hiragana や行
	'や': "や", 'ゆ': "や", 'よ': "や",
	// Hiragana ら行
	'ら': "ら", 'り': "ら", 'る': "ら", 'れ': "ら", 'ろ': "ら",
	// Hiragana わ行
	'わ': "わ", 'を': "わ", 'ん': "わ",

	// Katakana ア行
	'ア': "あ", 'イ': "あ", 'ウ': "あ", 'エ': "あ", 'オ': "あ",
	// Katakana カ行
	'カ': "か", 'キ': "か", 'ク': "か", 'ケ': "か", 'コ': "か",
	'ガ': "か", 'ギ': "か", 'グ': "か", 'ゲ': "か", 'ゴ': "か",
	// Katakana サ行
	'サ': "さ", 'シ': "さ", 'ス': "さ", 'セ': "さ", 'ソ': "さ",
	'ザ': "さ", 'ジ': "さ", 'ズ': "さ", 'ゼ': "さ", 'ゾ': "さ",
	// Katakana タ行
	'タ': "た", 'チ': "た", 'ツ': "た", 'テ': "た", 'ト': "た",
	'ダ': "た", 'ヂ': "た", 'ヅ': "た", 'デ': "た", 'ド': "た",
	// Katakana ナ行
	'ナ': "な", 'ニ': "な", 'ヌ': "な", 'ネ': "な", 'ノ': "な",
	// Katakana ハ行
	'ハ': "は", 'ヒ': "は", 'フ': "は", 'ヘ': "は", 'ホ': "は",
	'バ': "は", 'ビ': "は", 'ブ': "は", 'ベ': "は", 'ボ': "は",
	'パ': "は", 'ピ': "は", 'プ': "は", 'ペ': "は", 'ポ': "は",
	// Katakana マ行
	'マ': "ま", 'ミ': "ま", 'ム': "ま", 'メ': "ま", 'モ': "ま",
	// Katakana ヤ行
	'ヤ': "や", 'ユ': "や", 'ヨ': "や",
	// Katakana ラ行
	'ラ': "ら", 'リ': "ら", 'ル': "ら", 'レ': "ら", 'ロ': "ら",
	// Katakana ワ行
	'ワ': "わ", 'ヲ': "わ", 'ン': "わ",
}

// Kana to Romaji first letter mapping
var kanaToRomaji = map[rune]string{
	// あ行 -> A
	'あ': "A", 'い': "I", 'う': "U", 'え': "E", 'お': "O",
	'ア': "A", 'イ': "I", 'ウ': "U", 'エ': "E", 'オ': "O",
	// か行 -> K (が行 -> G)
	'か': "K", 'き': "K", 'く': "K", 'け': "K", 'こ': "K",
	'カ': "K", 'キ': "K", 'ク': "K", 'ケ': "K", 'コ': "K",
	'が': "G", 'ぎ': "G", 'ぐ': "G", 'げ': "G", 'ご': "G",
	'ガ': "G", 'ギ': "G", 'グ': "G", 'ゲ': "G", 'ゴ': "G",
	// さ行 -> S (ざ行 -> Z)
	'さ': "S", 'し': "S", 'す': "S", 'せ': "S", 'そ': "S",
	'サ': "S", 'シ': "S", 'ス': "S", 'セ': "S", 'ソ': "S",
	'ざ': "Z", 'じ': "Z", 'ず': "Z", 'ぜ': "Z", 'ぞ': "Z",
	'ザ': "Z", 'ジ': "Z", 'ズ': "Z", 'ゼ': "Z", 'ゾ': "Z",
	// た行 -> T (だ行 -> D)
	'た': "T", 'ち': "T", 'つ': "T", 'て': "T", 'と': "T",
	'タ': "T", 'チ': "T", 'ツ': "T", 'テ': "T", 'ト': "T",
	'だ': "D", 'ぢ': "D", 'づ': "D", 'で': "D", 'ど': "D",
	'ダ': "D", 'ヂ': "D", 'ヅ': "D", 'デ': "D", 'ド': "D",
	// な行 -> N
	'な': "N", 'に': "N", 'ぬ': "N", 'ね': "N", 'の': "N",
	'ナ': "N", 'ニ': "N", 'ヌ': "N", 'ネ': "N", 'ノ': "N",
	// は行 -> H (ば行 -> B, ぱ行 -> P)
	'は': "H", 'ひ': "H", 'ふ': "H", 'へ': "H", 'ほ': "H",
	'ハ': "H", 'ヒ': "H", 'フ': "H", 'ヘ': "H", 'ホ': "H",
	'ば': "B", 'び': "B", 'ぶ': "B", 'べ': "B", 'ぼ': "B",
	'バ': "B", 'ビ': "B", 'ブ': "B", 'ベ': "B", 'ボ': "B",
	'ぱ': "P", 'ぴ': "P", 'ぷ': "P", 'ぺ': "P", 'ぽ': "P",
	'パ': "P", 'ピ': "P", 'プ': "P", 'ペ': "P", 'ポ': "P",
	// ま行 -> M
	'ま': "M", 'み': "M", 'む': "M", 'め': "M", 'も': "M",
	'マ': "M", 'ミ': "M", 'ム': "M", 'メ': "M", 'モ': "M",
	// や行 -> Y
	'や': "Y", 'ゆ': "Y", 'よ': "Y",
	'ヤ': "Y", 'ユ': "Y", 'ヨ': "Y",
	// ら行 -> R
	'ら': "R", 'り': "R", 'る': "R", 'れ': "R", 'ろ': "R",
	'ラ': "R", 'リ': "R", 'ル': "R", 'レ': "R", 'ロ': "R",
	// わ行 -> W
	'わ': "W", 'を': "W", 'ん': "N",
	'ワ': "W", 'ヲ': "W", 'ン': "N",
}

// CalculateInitials computes both A-Z and Kana initials for a tab title
// based on the artist's origin country (from MusicBrainz).
//
// Returns:
//   - az: A-Z initial for EN/ZH UI (Pinyin/Romaji mapped to A-Z)
//   - kana: Kana initial for JA UI (あかさたな... or A-Z for Latin)
//
// Logic:
//   - Chinese (CN/TW/HK): az = Pinyin uppercase, kana = "#"
//   - Japanese (JP): az = Romaji uppercase, kana = Gojūon row
//   - Latin/English: az = uppercase A-Z, kana = same uppercase A-Z
//   - Fallback: Heuristic detection or "#" for both
func CalculateInitials(title string, originCountry string) (az string, kana string) {
	if title == "" {
		return "#", "#"
	}

	// Get first character
	firstChar := []rune(strings.TrimSpace(title))[0]

	// Normalize origin country
	originCountry = strings.ToUpper(strings.TrimSpace(originCountry))

	// Determine language based on origin country
	switch originCountry {
	case "CN", "TW", "HK": // Chinese
		return calculateChineseInitials(title)
	case "JP": // Japanese
		return calculateJapaneseInitials(title)
	case "US", "GB", "CA", "AU", "NZ", "IE": // English-speaking countries
		return calculateLatinInitials(title)
	default:
		// Fallback: Heuristic detection
		if isKana(firstChar) {
			return calculateJapaneseInitials(title)
		} else if isHanzi(firstChar) {
			return calculateChineseInitials(title)
		} else if isLatin(firstChar) {
			return calculateLatinInitials(title)
		}
		return "#", "#"
	}
}

// calculateChineseInitials extracts Pinyin initial for Chinese titles
func calculateChineseInitials(title string) (az string, kana string) {
	// Use go-pinyin to get Pinyin
	args := pinyin.NewArgs()
	args.Style = pinyin.FirstLetter // Get first letter only
	args.Fallback = func(r rune, a pinyin.Args) []string {
		// Fallback for non-Chinese characters
		if isLatin(r) {
			return []string{strings.ToUpper(string(r))}
		}
		return []string{"#"}
	}

	pinyinResult := pinyin.Pinyin(title, args)
	if len(pinyinResult) > 0 && len(pinyinResult[0]) > 0 {
		initial := strings.ToUpper(pinyinResult[0][0])
		if len(initial) > 0 && initial[0] >= 'A' && initial[0] <= 'Z' {
			return string(initial[0]), "#"
		}
	}

	return "#", "#"
}

// calculateJapaneseInitials extracts Kana row and Romaji initial for Japanese titles
func calculateJapaneseInitials(title string) (az string, kana string) {
	// Try to get reading using Kagome tokenizer
	t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		// Fallback to direct character mapping
		return calculateJapaneseInitialsFallback(title)
	}

	tokens := t.Tokenize(title)
	if len(tokens) == 0 {
		return calculateJapaneseInitialsFallback(title)
	}

	// Get the reading (yomi) of the first token
	firstToken := tokens[0]
	features := firstToken.Features()

	// IPA dict features: [品詞,品詞細分類1,品詞細分類2,品詞細分類3,活用型,活用形,原形,読み,発音]
	// Reading is at index 7, but not all tokens have all features
	if len(features) < 8 {
		return calculateJapaneseInitialsFallback(title)
	}

	reading := features[7]

	if reading == "*" || reading == "" {
		// No reading available, use fallback
		return calculateJapaneseInitialsFallback(title)
	}

	// Get first character of reading
	readingRunes := []rune(reading)
	if len(readingRunes) == 0 {
		return calculateJapaneseInitialsFallback(title)
	}

	firstKana := readingRunes[0]

	// Map to Romaji for A-Z
	if romaji, ok := kanaToRomaji[firstKana]; ok {
		az = romaji
	} else {
		az = "#"
	}

	// Map to Gojūon row for Kana
	if row, ok := kanaToRow[firstKana]; ok {
		kana = row
	} else {
		kana = "#"
	}

	return az, kana
}

// calculateJapaneseInitialsFallback uses direct character mapping when tokenizer fails
func calculateJapaneseInitialsFallback(title string) (az string, kana string) {
	firstChar := []rune(strings.TrimSpace(title))[0]

	// Map to Romaji for A-Z
	if romaji, ok := kanaToRomaji[firstChar]; ok {
		az = romaji
	} else if isLatin(firstChar) {
		az = strings.ToUpper(string(firstChar))
	} else {
		az = "#"
	}

	// Map to Gojūon row for Kana
	if row, ok := kanaToRow[firstChar]; ok {
		kana = row
	} else if isLatin(firstChar) {
		kana = strings.ToUpper(string(firstChar))
	} else {
		kana = "#"
	}

	return az, kana
}

// calculateLatinInitials extracts uppercase initial for Latin/English titles
func calculateLatinInitials(title string) (az string, kana string) {
	firstChar := []rune(strings.TrimSpace(title))[0]

	if isLatin(firstChar) {
		initial := strings.ToUpper(string(firstChar))
		return initial, initial // Same for both A-Z and Kana
	}

	return "#", "#"
}

// isKana checks if a character is Hiragana or Katakana
func isKana(r rune) bool {
	return (r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) // Katakana
}

// isHanzi checks if a character is a Chinese character (CJK Unified Ideographs)
func isHanzi(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x20000 && r <= 0x2A6DF) // CJK Extension B
}

// isLatin checks if a character is a Latin letter (A-Z, a-z)
func isLatin(r rune) bool {
	return unicode.IsLetter(r) && ((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'))
}

