package types

import (
	"errors"
	"fmt"
	"os"
)

// Locale constants
type Locale string

// Locale available in amazon API
// reference: https://webservices.amazon.com/paapi5/documentation/common-request-parameters.html#host-and-region
const (
	LocaleAustralia          Locale = "www.amazon.com.au" // Australia
	LocaleBrazil             Locale = "www.amazon.com.br" // Brazil
	LocaleCanada             Locale = "www.amazon.ca"     // Canada
	LocaleFrance             Locale = "www.amazon.fr"     // France
	LocaleGermany            Locale = "www.amazon.de"     // Germany
	LocaleIndia              Locale = "www.amazon.in"     // India
	LocaleItaly              Locale = "www.amazon.it"     // Italy
	LocaleJapan              Locale = "www.amazon.co.jp"  // Japan
	LocaleMexico             Locale = "www.amazon.com.mx" // Mexico
	LocaleNetherlands        Locale = "www.amazon.nl"     // Netherlands
	LocaleSingapore          Locale = "www.amazon.sg"     // Singapore
	LocaleSaudiArabia        Locale = "www.amazon.sa"     // Saudi Arabia
	LocaleSpain              Locale = "www.amazon.es"     // Spain
	LocaleSweden             Locale = "www.amazon.se"     // Sweden
	LocaleTurkey             Locale = "www.amazon.com.tr" // Turkey
	LocaleUnitedArabEmirates Locale = "www.amazon.ae"     // United Arab Emirates
	LocaleUnitedKingdom      Locale = "www.amazon.co.uk"  // United Kingdom
	LocaleUnitedStates       Locale = "www.amazon.com"    // United States
)

// reference: https://webservices.amazon.com/paapi5/documentation/common-request-parameters.html#host-and-region
var localeHostMap = map[Locale]string{
	LocaleAustralia:          "webservices.amazon.com.au",
	LocaleBrazil:             "webservices.amazon.com.br",
	LocaleCanada:             "webservices.amazon.ca",
	LocaleFrance:             "webservices.amazon.fr",
	LocaleGermany:            "webservices.amazon.de",
	LocaleIndia:              "webservices.amazon.in",
	LocaleItaly:              "webservices.amazon.it",
	LocaleJapan:              "webservices.amazon.co.jp",
	LocaleMexico:             "webservices.amazon.com.mx",
	LocaleNetherlands:        "webservices.amazon.nl",
	LocaleSingapore:          "webservices.amazon.sg",
	LocaleSaudiArabia:        "webservices.amazon.sa",
	LocaleSpain:              "webservices.amazon.es",
	LocaleSweden:             "webservices.amazon.se",
	LocaleTurkey:             "webservices.amazon.com.tr",
	LocaleUnitedArabEmirates: "webservices.amazon.ae",
	LocaleUnitedKingdom:      "webservices.amazon.co.uk",
	LocaleUnitedStates:       "webservices.amazon.com",
}

// reference: https://webservices.amazon.com/paapi5/documentation/common-request-parameters.html#host-and-region
var localRegionMap = map[Locale]string{
	LocaleAustralia:          "us-west-2",
	LocaleBrazil:             "us-east-1",
	LocaleCanada:             "us-east-1",
	LocaleFrance:             "eu-west-1",
	LocaleGermany:            "eu-west-1",
	LocaleIndia:              "eu-west-1",
	LocaleItaly:              "eu-west-1",
	LocaleJapan:              "us-west-2",
	LocaleMexico:             "us-east-1",
	LocaleNetherlands:        "eu-west-1",
	LocaleSingapore:          "us-west-2",
	LocaleSaudiArabia:        "eu-west-1",
	LocaleSpain:              "eu-west-1",
	LocaleSweden:             "eu-west-1",
	LocaleTurkey:             "eu-west-1",
	LocaleUnitedArabEmirates: "eu-west-1",
	LocaleUnitedKingdom:      "eu-west-1",
	LocaleUnitedStates:       "us-east-1",
}

// Setting assocoate tags based on locale
var localeAssociateMap = map[Locale]string{
	LocaleUnitedStates:  "cosmopolitan-20",
	LocaleUnitedKingdom: "hearstmagazin-21",
	LocaleNetherlands:   "hearstnl03-21",
	LocaleJapan:         "hearstjapan-22",
	LocaleItaly:         "hearstmagaz03-21",
	LocaleSpain:         "hearstmagaz0a-21",
}

// Set different PAPI access key for different regions
var localePAPIAccessKeyMap = map[Locale]string{
	LocaleUnitedStates:  os.Getenv("PAAPI_ACCESS_KEY"),
	LocaleUnitedKingdom: os.Getenv("PAAPI_ACCESS_KEY"),
	LocaleNetherlands:   os.Getenv("NL_PAAPI_ACCESS_KEY"),
	LocaleJapan:         os.Getenv("JP_PAAPI_ACCESS_KEY"),
	LocaleItaly:         os.Getenv("PAAPI_ACCESS_KEY"),
	LocaleSpain:         os.Getenv("PAAPI_ACCESS_KEY"),
}

// mapping of locale to secret keys taken from environment
var localePAPISecretMap = map[Locale]string{
	LocaleUnitedStates:  os.Getenv("PAAPI_SECRET_KEY"),
	LocaleUnitedKingdom: os.Getenv("PAAPI_SECRET_KEY"),
	LocaleNetherlands:   os.Getenv("NL_PAAPI_SECRET_KEY"),
	LocaleJapan:         os.Getenv("JP_PAAPI_SECRET_KEY"),
	LocaleItaly:         os.Getenv("PAAPI_SECRET_KEY"),
	LocaleSpain:         os.Getenv("PAAPI_SECRET_KEY"),
}

// mapping of locale to sem3 geo ID
var localeGeoIDMap = map[Locale]int{
	LocaleUnitedStates:  1,
	LocaleUnitedKingdom: 2,
	LocaleNetherlands:   29,
	LocaleJapan:         17,
	LocaleItaly:         28,
	LocaleSpain:         30,
}

// localFromHost get local variable from URL
func NewLocale(host string) (Locale, error) {
	localeHost := Locale(host)
	if _, ok := localeHostMap[localeHost]; ok {
		return localeHost, nil
	}
	return "", fmt.Errorf("unsupported host: %s", host)
}

// Host returns API endpoint for locale
func (locale Locale) Host() string {
	return localeHostMap[locale]
}

// Scheme returns API endpoint for locale
func (locale Locale) Scheme() string {
	return "https"
}

// Region returns region for locale
func (locale Locale) Region() string {
	return localRegionMap[locale]
}

// AssociateTag returns associate tag based on locale
func (locale Locale) AssociateTag() (string, error) {
	associateTag := localeAssociateMap[locale]
	if associateTag == "" {
		return "", errors.New("amazon region/tag not supported")
	}
	return associateTag, nil
}

// SecretKey returns PAAPI access key
func (locale Locale) AccessKey() (string, error) {
	accessKey := localePAPIAccessKeyMap[locale]
	if accessKey == "" {
		return "", errors.New("access key not found")
	}
	return accessKey, nil
}

// SecretKey returns PAAPI secret key
func (locale Locale) SecretKey() (string, error) {
	secretKey := localePAPISecretMap[locale]
	if secretKey == "" {
		return "", errors.New("secret key not found")
	}
	return secretKey, nil
}

// MarketPlace returns marketplace for locale
func (locale Locale) MarketPlace() string {
	return string(locale)
}

// GeoID returns sem3 geo ID for the given region
func (locale Locale) GeoID() (int, error) {
	geoID := localeGeoIDMap[locale]
	if geoID == 0 {
		return 0, errors.New("geo ID not found")
	}
	return geoID, nil
}
