/** ISO 639-1 two-letter primary language subtag (lowercase). */
type LangCodeBase = Lowercase<
  | 'aa' // Afar
  | 'ab' // Abkhazian
  | 'ae' // Avestan
  | 'af' // Afrikaans
  | 'ak' // Akan
  | 'am' // Amharic
  | 'an' // Aragonese
  | 'ar' // Arabic
  | 'as' // Assamese
  | 'av' // Avaric
  | 'ay' // Aymara
  | 'az' // Azerbaijani
  | 'ba' // Bashkir
  | 'be' // Belarusian
  | 'bg' // Bulgarian
  | 'bi' // Bislama
  | 'bm' // Bambara
  | 'bn' // Bengali
  | 'bo' // Tibetan
  | 'br' // Breton
  | 'bs' // Bosnian
  | 'ca' // Catalan
  | 'ce' // Chechen
  | 'ch' // Chamorro
  | 'co' // Corsican
  | 'cr' // Cree
  | 'cs' // Czech
  | 'cu' // Church Slavonic
  | 'cv' // Chuvash
  | 'cy' // Welsh
  | 'da' // Danish
  | 'de' // German
  | 'dv' // Divehi
  | 'dz' // Dzongkha
  | 'ee' // Ewe
  | 'el' // Greek
  | 'en' // English
  | 'eo' // Esperanto
  | 'es' // Spanish
  | 'et' // Estonian
  | 'eu' // Basque
  | 'fa' // Persian
  | 'ff' // Fulah
  | 'fi' // Finnish
  | 'fj' // Fijian
  | 'fo' // Faroese
  | 'fr' // French
  | 'fy' // Western Frisian
  | 'ga' // Irish
  | 'gd' // Gaelic
  | 'gl' // Galician
  | 'gn' // Guarani
  | 'gu' // Gujarati
  | 'gv' // Manx
  | 'ha' // Hausa
  | 'he' // Hebrew
  | 'hi' // Hindi
  | 'ho' // Hiri Motu
  | 'hr' // Croatian
  | 'ht' // Haitian
  | 'hu' // Hungarian
  | 'hy' // Armenian
  | 'hz' // Herero
  | 'ia' // Interlingua
  | 'id' // Indonesian
  | 'ie' // Interlingue
  | 'ig' // Igbo
  | 'ii' // Sichuan Yi
  | 'ik' // Inupiaq
  | 'io' // Ido
  | 'is' // Icelandic
  | 'it' // Italian
  | 'iu' // Inuktitut
  | 'ja' // Japanese
  | 'jv' // Javanese
  | 'ka' // Georgian
  | 'kg' // Kongo
  | 'ki' // Kikuyu
  | 'kj' // Kuanyama
  | 'kk' // Kazakh
  | 'kl' // Kalaallisut
  | 'km' // Central Khmer
  | 'kn' // Kannada
  | 'ko' // Korean
  | 'kr' // Kanuri
  | 'ks' // Kashmiri
  | 'ku' // Kurdish
  | 'kv' // Komi
  | 'kw' // Cornish
  | 'ky' // Kyrgyz
  | 'la' // Latin
  | 'lb' // Luxembourgish
  | 'lg' // Ganda
  | 'li' // Limburgan
  | 'ln' // Lingala
  | 'lo' // Lao
  | 'lt' // Lithuanian
  | 'lu' // Luba-Katanga
  | 'lv' // Latvian
  | 'mg' // Malagasy
  | 'mh' // Marshallese
  | 'mi' // Maori
  | 'mk' // Macedonian
  | 'ml' // Malayalam
  | 'mn' // Mongolian
  | 'mr' // Marathi
  | 'ms' // Malay
  | 'mt' // Maltese
  | 'my' // Burmese
  | 'na' // Nauru
  | 'nb' // Norwegian Bokmål
  | 'nd' // North Ndebele
  | 'ne' // Nepali
  | 'ng' // Ndonga
  | 'nl' // Dutch
  | 'nn' // Norwegian Nynorsk
  | 'no' // Norwegian
  | 'nr' // South Ndebele
  | 'nv' // Navajo
  | 'ny' // Chichewa
  | 'oc' // Occitan
  | 'oj' // Ojibwa
  | 'om' // Oromo
  | 'or' // Oriya
  | 'os' // Ossetian
  | 'pa' // Punjabi
  | 'pi' // Pali
  | 'pl' // Polish
  | 'ps' // Pashto
  | 'pt' // Portuguese
  | 'qu' // Quechua
  | 'rm' // Romansh
  | 'rn' // Rundi
  | 'ro' // Romanian
  | 'ru' // Russian
  | 'rw' // Kinyarwanda
  | 'sa' // Sanskrit
  | 'sc' // Sardinian
  | 'sd' // Sindhi
  | 'se' // Northern Sami
  | 'sg' // Sango
  | 'si' // Sinhala
  | 'sk' // Slovak
  | 'sl' // Slovenian
  | 'sm' // Samoan
  | 'sn' // Shona
  | 'so' // Somali
  | 'sq' // Albanian
  | 'sr' // Serbian
  | 'ss' // Swati
  | 'st' // Southern Sotho
  | 'su' // Sundanese
  | 'sv' // Swedish
  | 'sw' // Swahili
  | 'ta' // Tamil
  | 'te' // Telugu
  | 'tg' // Tajik
  | 'th' // Thai
  | 'ti' // Tigrinya
  | 'tk' // Turkmen
  | 'tl' // Tagalog
  | 'tn' // Tswana
  | 'to' // Tonga
  | 'tr' // Turkish
  | 'ts' // Tsonga
  | 'tt' // Tatar
  | 'tw' // Twi
  | 'ty' // Tahitian
  | 'ug' // Uighur
  | 'uk' // Ukrainian
  | 'ur' // Urdu
  | 'uz' // Uzbek
  | 've' // Venda
  | 'vi' // Vietnamese
  | 'vo' // Volapük
  | 'wa' // Walloon
  | 'wo' // Wolof
  | 'xh' // Xhosa
  | 'yi' // Yiddish
  | 'yo' // Yoruba
  | 'za' // Zhuang
  | 'zh' // Chinese
  | 'zu' // Zulu
>

/**
 * RFC 5646 (BCP 47) language tag. Primary subtag must be a known ISO 639-1 code;
 * optional script, region, and variant subtags follow as hyphen-separated **LOWERCASE** strings.
 *
 * @example `en` | `en-us` | `zh-hant` | `zh-hant-tw` | `sr-latn-rs`
 */
export type LangCode = LangCodeBase | `${LangCodeBase}-${Lowercase<string>}`

/**
 * Returns true if the given string is a valid language code (must be lowercase and follow the BCP 47 format).
 *
 * @example
 * ```ts
 * isLangCode('en') // true
 * isLangCode('en-us') // true
 * isLangCode('zh-hant') // true
 * isLangCode('zh-hant-tw') // true
 * isLangCode('sr-latn-rs') // true
 * isLangCode('EN') // false (must be lowercase)
 * isLangCode('en-US') // false (must be lowercase)
 * ```
 */
export const isLangCode = (v: string): v is LangCode => /^[a-z]{2}(-[a-z0-9]{1,8})*$/.test(v)
