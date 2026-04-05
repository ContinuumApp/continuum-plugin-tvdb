package provider

// iso639_1To2 maps ISO 639-1 two-letter codes to ISO 639-2/B three-letter
// codes used by the TVDB v4 API translation endpoints.
var iso639_1To2 = map[string]string{
	"en": "eng", "ja": "jpn", "fr": "fra", "de": "deu",
	"es": "spa", "it": "ita", "pt": "por", "zh": "zho",
	"ko": "kor", "ru": "rus", "nl": "nld", "pl": "pol",
	"sv": "swe", "da": "dan", "no": "nor", "fi": "fin",
	"tr": "tur", "cs": "ces", "hu": "hun", "he": "heb",
	"ar": "ara", "th": "tha", "id": "ind", "vi": "vie",
	"uk": "ukr", "ro": "ron", "bg": "bul", "hr": "hrv",
	"sk": "slk", "el": "ell", "sl": "slv", "ms": "msa",
	"hi": "hin", "ta": "tam", "te": "tel", "bn": "ben",
	"fa": "fas",
}

// toLang3 converts a language tag (ISO 639-1 or already 3-letter) to the
// ISO 639-2 three-letter code the TVDB API uses in URL paths.
// Input may include region suffixes (e.g. "fr-CA" → "fra").
// Returns "eng" for unknown or empty inputs.
func toLang3(lang string) string {
	norm := normalizeLanguageTag(lang)
	if len(norm) == 3 {
		return norm
	}
	if code, ok := iso639_1To2[norm]; ok {
		return code
	}
	return "eng"
}

// isEnglish returns true if the language resolves to English.
func isEnglish(lang string) bool {
	return toLang3(lang) == "eng"
}
