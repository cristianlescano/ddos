package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/474420502/gcurl"
	"github.com/leekchan/accounting"
)

// ---------------------------------------------------------------------------
// Structs
// ---------------------------------------------------------------------------

// requestConfig holds all the parameters needed to make an HTTP request
type requestConfig struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    io.Reader
}

// browserProfile represents a coherent set of browser identification headers
type browserProfile struct {
	userAgent string
	secChUa   string
	platform  string
	mobile    string
}

// navigationProfile defines the "context" of the request (page load, asset, API, etc.)
type navigationProfile struct {
	name            string
	secFetchDest    string
	secFetchMode    string
	secFetchSite    string
	secFetchUser    string // empty if not applicable
	upgradeInsecure bool
	acceptHeader    string
	refererType     string // "none", "same-origin", "cross-site"
}

// ---------------------------------------------------------------------------
// Browser profiles pool (60+ real browser signatures)
// ---------------------------------------------------------------------------

var browserProfiles = []browserProfile{
	// Chrome 148
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36", `"Chromium";v="148", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36 Edg/148.0.0.0", `"Microsoft Edge";v="148", "Chromium";v="148", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36", `"Chromium";v="148", "Not.A/Brand";v="8"`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36", `"Chromium";v="148", "Not.A/Brand";v="8"`, `"Linux"`, "?0"},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36", `"Chromium";v="148", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},

	// Chrome 147
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36", `"Chromium";v="147", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36", `"Chromium";v="147", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36", `"Chromium";v="147", "Not.A/Brand";v="8"`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36", `"Chromium";v="147", "Not.A/Brand";v="8"`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36", `"Chromium";v="147", "Not.A/Brand";v="8"`, `"Linux"`, "?0"},
	{"Mozilla/5.0 (X11; Ubuntu; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36", `"Chromium";v="147", "Not.A/Brand";v="8"`, `"Linux"`, "?0"},

	// Chrome 146
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36", `"Chromium";v="146", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36", `"Chromium";v="146", "Not.A/Brand";v="8"`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36", `"Chromium";v="146", "Not.A/Brand";v="8"`, `"Linux"`, "?0"},

	// Chrome Android / iOS
	{"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Mobile Safari/537.36", `"Chromium";v="148", "Not.A/Brand";v="8"`, `"Android"`, "?1"},
	{"Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Mobile Safari/537.36", `"Chromium";v="147", "Not.A/Brand";v="8"`, `"Android"`, "?1"},
	{"Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Mobile Safari/537.36", `"Chromium";v="146", "Not.A/Brand";v="8"`, `"Android"`, "?1"},
	{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/148.0.0.0 Mobile/15E148 Safari/604.1", `"Chromium";v="148", "Not.A/Brand";v="8"`, `"iOS"`, "?1"},
	{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/147.0.0.0 Mobile/15E148 Safari/604.1", `"Chromium";v="147", "Not.A/Brand";v="8"`, `"iOS"`, "?1"},
	{"Mozilla/5.0 (iPad; CPU OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/146.0.0.0 Mobile/15E148 Safari/604.1", `"Chromium";v="146", "Not.A/Brand";v="8"`, `"iOS"`, "?0"},

	// Brave
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36", `"Brave";v="148", "Not.A/Brand";v="8", "Chromium";v="148"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36", `"Brave";v="148", "Not.A/Brand";v="8", "Chromium";v="148"`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36", `"Brave";v="148", "Not.A/Brand";v="8", "Chromium";v="148"`, `"Linux"`, "?0"},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36", `"Brave";v="147", "Not.A/Brand";v="8", "Chromium";v="147"`, `"Windows"`, "?0"},

	// Edge
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36 Edg/148.0.0.0", `"Microsoft Edge";v="148", "Chromium";v="148", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36 Edg/147.0.0.0", `"Microsoft Edge";v="147", "Chromium";v="147", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36 Edg/148.0.0.0", `"Microsoft Edge";v="148", "Chromium";v="148", "Not.A/Brand";v="8"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36 Edg/148.0.0.0", `"Microsoft Edge";v="148", "Chromium";v="148", "Not.A/Brand";v="8"`, `"macOS"`, "?0"},

	// Firefox
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0", `""`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:139.0) Gecko/20100101 Firefox/139.0", `""`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0", `""`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:140.0) Gecko/20100101 Firefox/140.0", `""`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:139.0) Gecko/20100101 Firefox/139.0", `""`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:138.0) Gecko/20100101 Firefox/138.0", `""`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0", `""`, `"Linux"`, "?0"},
	{"Mozilla/5.0 (X11; Linux x86_64; rv:139.0) Gecko/20100101 Firefox/139.0", `""`, `"Linux"`, "?0"},
	{"Mozilla/5.0 (X11; Linux x86_64; rv:138.0) Gecko/20100101 Firefox/138.0", `""`, `"Linux"`, "?0"},
	{"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0", `""`, `"Linux"`, "?0"},

	// Firefox Mobile
	{"Mozilla/5.0 (Android 14; Mobile; rv:140.0) Gecko/140.0 Firefox/140.0", `""`, `"Android"`, "?1"},
	{"Mozilla/5.0 (Android 13; Mobile; rv:139.0) Gecko/139.0 Firefox/139.0", `""`, `"Android"`, "?1"},

	// Safari macOS
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15", `""`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15", `""`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15", `""`, `"macOS"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15", `""`, `"macOS"`, "?0"},

	// Safari iOS
	{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1", `""`, `"iOS"`, "?1"},
	{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1", `""`, `"iOS"`, "?1"},
	{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1", `""`, `"iOS"`, "?1"},
	{"Mozilla/5.0 (iPad; CPU OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1", `""`, `"iOS"`, "?0"},

	// Opera
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36 OPR/106.0.0.0", `"Chromium";v="148", "Not.A/Brand";v="8", "Opera";v="106"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36 OPR/105.0.0.0", `"Chromium";v="147", "Not.A/Brand";v="8", "Opera";v="105"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36 OPR/106.0.0.0", `"Chromium";v="148", "Not.A/Brand";v="8", "Opera";v="106"`, `"macOS"`, "?0"},

	// Vivaldi
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36 Vivaldi/6.9.3447.37", `"Chromium";v="148", "Not.A/Brand";v="8", "Vivaldi";v="6.9"`, `"Windows"`, "?0"},
	{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36 Vivaldi/6.8.3381.44", `"Chromium";v="147", "Not.A/Brand";v="8", "Vivaldi";v="6.8"`, `"Linux"`, "?0"},

	// Samsung Internet
	{"Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/25.0 Chrome/121.0.0.0 Mobile Safari/537.36", `"Chromium";v="121", "Not.A/Brand";v="8", "Samsung Internet";v="25.0"`, `"Android"`, "?1"},
	{"Mozilla/5.0 (Linux; Android 13; SM-G998B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/24.0 Chrome/120.0.0.0 Mobile Safari/537.36", `"Chromium";v="120", "Not.A/Brand";v="8", "Samsung Internet";v="24.0"`, `"Android"`, "?1"},
}

// ---------------------------------------------------------------------------
// Navigation profiles (realistic request contexts)
// ---------------------------------------------------------------------------

var navigationProfiles = []navigationProfile{
	{
		name:            "page_navigate",
		secFetchDest:    "document",
		secFetchMode:    "navigate",
		secFetchSite:    "none",
		secFetchUser:    "?1",
		upgradeInsecure: true,
		acceptHeader:    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		refererType:     "none",
	},
	{
		name:            "page_reload",
		secFetchDest:    "document",
		secFetchMode:    "navigate",
		secFetchSite:    "same-origin",
		secFetchUser:    "?1",
		upgradeInsecure: true,
		acceptHeader:    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		refererType:     "same-origin",
	},
	{
		name:         "ajax_api",
		secFetchDest: "empty",
		secFetchMode: "cors",
		secFetchSite: "same-origin",
		secFetchUser: "",
		acceptHeader: "application/json, text/plain, */*",
		refererType:  "same-origin",
	},
	{
		name:         "image_load",
		secFetchDest: "image",
		secFetchMode: "no-cors",
		secFetchSite: "same-origin",
		secFetchUser: "",
		acceptHeader: "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8",
		refererType:  "same-origin",
	},
	{
		name:         "script_load",
		secFetchDest: "script",
		secFetchMode: "no-cors",
		secFetchSite: "cross-site",
		secFetchUser: "",
		acceptHeader: "*/*",
		refererType:  "cross-site",
	},
	{
		name:         "stylesheet",
		secFetchDest: "style",
		secFetchMode: "no-cors",
		secFetchSite: "cross-site",
		secFetchUser: "",
		acceptHeader: "text/css,*/*;q=0.1",
		refererType:  "cross-site",
	},
}

// ---------------------------------------------------------------------------
// Accept-Language pool
// ---------------------------------------------------------------------------

var acceptLanguages = []string{
	"es-419,es;q=0.9",
	"es-AR,es;q=0.9,en;q=0.8",
	"es-ES,es;q=0.9",
	"en-US,en;q=0.9,es;q=0.8",
	"en-GB,en;q=0.9",
	"es-MX,es;q=0.9",
	"es-CO,es;q=0.9,en;q=0.8",
	"pt-BR,pt;q=0.9,en;q=0.8",
	"en-US,en;q=0.9",
	"es-CL,es;q=0.9,en;q=0.8",
	"es-PE,es;q=0.9",
	"en-CA,en;q=0.9,fr;q=0.8",
}

// ---------------------------------------------------------------------------
// Cache-buster parameter names
// ---------------------------------------------------------------------------

var cacheBusterNames = []string{"_", "t", "v", "cb", "_cb", "nocache", "r", "ts"}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// chance returns true with probability percent/100.
func chance(percent int) bool {
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	return rand.Intn(100) < percent
}

// selectNavigationProfile picks a random navigation context.
func selectNavigationProfile() navigationProfile {
	return navigationProfiles[rand.Intn(len(navigationProfiles))]
}

// generateReferer creates a realistic referer based on the navigation context.
// refererType: "none" → empty, "same-origin" → same domain, "cross-site" → different domain or empty
func generateReferer(targetURL string, refererType string) string {
	if refererType == "none" {
		return ""
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}

	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}

	if refererType == "same-origin" {
		paths := []string{
			"/",
			"/home",
			"/index.html",
			"/search",
			"/products",
			"/category/item",
			"/blog/post-1",
			"/about",
			"/contact",
			"/login",
			"/dashboard",
			"/page/1",
			"/article/123",
			"/user/profile",
		}
		return scheme + "://" + u.Host + paths[rand.Intn(len(paths))]
	}

	// cross-site: sometimes empty, sometimes a realistic external referrer
	if chance(40) {
		return ""
	}

	externalDomains := []string{
		"google.com",
		"www.google.com",
		"bing.com",
		"www.bing.com",
		"twitter.com",
		"x.com",
		"facebook.com",
		"www.facebook.com",
		"reddit.com",
		"news.ycombinator.com",
		"linkedin.com",
		"duckduckgo.com",
	}

	extPaths := []string{
		"/search?q=" + url.QueryEscape(u.Host),
		"/",
		"/feed",
		"/timeline",
	}

	domain := externalDomains[rand.Intn(len(externalDomains))]
	return "https://" + domain + extPaths[rand.Intn(len(extPaths))]
}

// buildCacheBuster adds realistic cache-busting query parameters.
//   - 20%: no cache-buster
//   - 65%: 1 parameter
//   - 15%: 2 parameters
func buildCacheBuster(rawURL string) string {
	r := rand.Intn(100)
	if r < 20 {
		return rawURL // no cache-buster
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	q := u.Query()

	if r < 85 {
		// 1 parameter (65% total)
		name := cacheBusterNames[rand.Intn(len(cacheBusterNames))]
		if chance(50) {
			q.Set(name, strconv.FormatInt(time.Now().UnixMilli(), 10))
		} else {
			q.Set(name, strconv.Itoa(rand.Intn(1000000000)))
		}
	} else {
		// 2 parameters (15% total)
		n1 := cacheBusterNames[rand.Intn(len(cacheBusterNames))]
		n2 := cacheBusterNames[rand.Intn(len(cacheBusterNames))]
		if n1 == n2 {
			n2 = cacheBusterNames[(rand.Intn(len(cacheBusterNames)-1)+1)%len(cacheBusterNames)]
		}
		q.Set(n1, strconv.FormatInt(time.Now().UnixMilli(), 10))
		q.Set(n2, strconv.Itoa(rand.Intn(1000000000)))
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// ---------------------------------------------------------------------------
// Header rotation
// ---------------------------------------------------------------------------

var rotatableHeaders = []string{
	"user-agent",
	"sec-ch-ua",
	"sec-ch-ua-mobile",
	"sec-ch-ua-platform",
	"accept-language",
	"sec-gpc",
	"accept",
	"accept-encoding",
	"referer",
	"sec-fetch-dest",
	"sec-fetch-mode",
	"sec-fetch-site",
	"sec-fetch-user",
	"upgrade-insecure-requests",
	"dnt",
}

func isRotatable(header string) bool {
	lower := strings.ToLower(header)
	for _, h := range rotatableHeaders {
		if lower == h {
			return true
		}
	}
	return false
}

func rotateHeaders(cfg requestConfig) requestConfig {
	rotated := requestConfig{
		URL:     cfg.URL,
		Method:  cfg.Method,
		Headers: make(map[string]string),
		Body:    cfg.Body,
	}

	// Copy all non-rotatable headers as-is
	for k, v := range cfg.Headers {
		if !isRotatable(k) {
			rotated.Headers[k] = v
		}
	}

	// Select profiles
	browser := browserProfiles[rand.Intn(len(browserProfiles))]
	nav := selectNavigationProfile()

	// Browser identification headers
	rotated.Headers["User-Agent"] = browser.userAgent
	if browser.secChUa != `""` {
		rotated.Headers["sec-ch-ua"] = browser.secChUa
	}
	rotated.Headers["sec-ch-ua-mobile"] = browser.mobile
	rotated.Headers["sec-ch-ua-platform"] = browser.platform

	// Accept-Language
	rotated.Headers["Accept-Language"] = acceptLanguages[rand.Intn(len(acceptLanguages))]

	// Accept (from navigation profile)
	rotated.Headers["Accept"] = nav.acceptHeader

	// Accept-Encoding (varied like real browsers)
	encodings := []string{
		"gzip, deflate, br",
		"gzip, deflate, br, zstd",
		"gzip, deflate",
	}
	rotated.Headers["Accept-Encoding"] = encodings[rand.Intn(len(encodings))]

	// Sec-Fetch headers (from navigation profile)
	rotated.Headers["Sec-Fetch-Dest"] = nav.secFetchDest
	rotated.Headers["Sec-Fetch-Mode"] = nav.secFetchMode
	rotated.Headers["Sec-Fetch-Site"] = nav.secFetchSite
	if nav.secFetchUser != "" {
		rotated.Headers["Sec-Fetch-User"] = nav.secFetchUser
	}

	// Upgrade-Insecure-Requests (only for page navigations, random)
	if nav.upgradeInsecure && chance(50) {
		rotated.Headers["Upgrade-Insecure-Requests"] = "1"
	}

	// Referer: MUST be coherent with Sec-Fetch-Site
	referer := generateReferer(cfg.URL, nav.refererType)
	if referer != "" {
		rotated.Headers["Referer"] = referer
	}

	// Optional headers (random presence)
	if chance(70) {
		rotated.Headers["sec-gpc"] = "1"
	}
	if chance(25) {
		rotated.Headers["DNT"] = "1"
	}

	return rotated
}

// ---------------------------------------------------------------------------
// Parse curl
// ---------------------------------------------------------------------------

func parseCurl(input string) (requestConfig, error) {
	input = strings.ReplaceAll(input, "\\\n", " ")
	input = strings.TrimSpace(input)

	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "curl ") {
		trimmed = trimmed[5:]
	} else if trimmed == "curl" {
		return requestConfig{}, fmt.Errorf("comando curl vacío")
	}

	parsed, err := gcurl.Parse("curl " + trimmed)
	if err != nil {
		return requestConfig{}, fmt.Errorf("error parseando curl: %v", err)
	}

	cfg := requestConfig{
		Headers: make(map[string]string),
	}

	if parsed.ParsedURL != nil {
		cfg.URL = parsed.ParsedURL.String()
	}

	cfg.Method = parsed.Method
	if cfg.Method == "" {
		if parsed.Body != nil && parsed.Body.Len() > 0 {
			cfg.Method = "POST"
		} else {
			cfg.Method = "GET"
		}
	}

	for k, v := range parsed.Header {
		if len(v) > 0 {
			cfg.Headers[k] = v[0]
		}
	}

	if len(parsed.Cookies) > 0 {
		cookieParts := []string{}
		for _, c := range parsed.Cookies {
			cookieParts = append(cookieParts, c.Name+"="+c.Value)
		}
		if _, exists := cfg.Headers["Cookie"]; !exists {
			cfg.Headers["Cookie"] = strings.Join(cookieParts, "; ")
		}
	}

	if parsed.Body != nil && parsed.Body.Len() > 0 {
		cfg.Body = parsed.Body
	}

	return cfg, nil
}

// ---------------------------------------------------------------------------
// Fetch
// ---------------------------------------------------------------------------

func fetch(cfg requestConfig, ch chan bool, sleep int, sizeMB chan float64) {
	reqCfg := rotateHeaders(cfg)

	client := &http.Client{}

	url := reqCfg.URL
	if reqCfg.Method == "GET" || reqCfg.Method == "" {
		url = buildCacheBuster(url)
	}

	req, err := http.NewRequest(reqCfg.Method, url, reqCfg.Body)
	if err != nil {
		ch <- false
		return
	}

	for k, v := range reqCfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		ch <- false
		return
	}
	defer resp.Body.Close()

	// Jitter: ±25% around the base sleep value
	if sleep > 0 {
		jitter := rand.Intn(sleep/2 + 1)
		actualSleep := sleep - sleep/4 + jitter
		time.Sleep(time.Duration(actualSleep) * time.Millisecond)
	}

	body, _ := io.ReadAll(resp.Body)

	size := len(body)
	sizeKB := float64(size) / 1024.0
	sizeMB <- sizeKB // envía tamaño en KB

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		ch <- true
	} else {
		ch <- false
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("modo (manual/auto): ")
	mode := readLine(reader)
	mode = strings.ToLower(mode)
	if mode != "auto" {
		mode = "manual"
	}

	var cfg requestConfig

	if mode == "auto" {
		fmt.Println("Modo automático: pegá el comando curl completo.")
		fmt.Print("curl> ")
		curlInput := readLine(reader)

		var err error
		cfg, err = parseCurl(curlInput)
		if err != nil {
			fmt.Printf("Error parseando curl: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("URL: %s\n", cfg.URL)
		fmt.Printf("Method: %s\n", cfg.Method)
		fmt.Printf("Headers: %d\n", len(cfg.Headers))
	} else {
		fmt.Print("url: ")
		urlScan := readLine(reader)

		fmt.Print("cookies (presiona Enter si no hay): ")
		cookiesInput := readLine(reader)

		cfg = requestConfig{
			URL:     urlScan,
			Method:  "GET",
			Headers: make(map[string]string),
		}

		if cookiesInput != "" {
			cfg.Headers["Cookie"] = cookiesInput
		}
	}

	fmt.Print("nro rutinas: ")
	numGoroutinesInput := readLine(reader)
	numGoroutines, err := strconv.Atoi(numGoroutinesInput)
	if err != nil || numGoroutines < 1 {
		fmt.Println("El número de rutinas debe ser al menos 1")
		os.Exit(1)
	}

	fmt.Print("sleep (ms): ")
	numSleepInput := readLine(reader)
	numSleep, err := strconv.Atoi(numSleepInput)
	if err != nil || numSleep < 0 {
		fmt.Println("El sleep debe ser un número válido >= 0")
		os.Exit(1)
	}

	ch := make(chan bool)
	sizeMB := make(chan float64)

	for i := 0; i < numGoroutines; i++ {
		go fetch(cfg, ch, numSleep, sizeMB)
	}

	ac := accounting.Accounting{
		Symbol:    "",
		Precision: 0,
		Thousand:  ".",
		Decimal:   "",
	}

	totalRequests := 0
	totalRequestsErr := 0
	totalSize := 0.0
	for {
		select {
		case success := <-ch:
			if success {
				totalRequests++
			} else {
				totalRequestsErr++
			}
			go fetch(cfg, ch, numSleep, sizeMB)

		case size := <-sizeMB:
			totalSize += size
		}

		sizeProm := 0.0
		if totalRequests > 0 {
			sizeProm = totalSize / float64(totalRequests)
		}

		// Porcentaje de acierto
		totalAll := totalRequests + totalRequestsErr
		hitRate := 0.0
		if totalAll > 0 {
			hitRate = float64(totalRequests) / float64(totalAll) * 100
		}

		// Formatear transferido: totalSize está en KB
		var transferStr string
		totalGB := totalSize / 1024.0 / 1024.0
		totalMB := totalSize / 1024.0
		if totalGB >= 1.0 {
			transferStr = fmt.Sprintf("%.2f GB", totalGB)
		} else {
			transferStr = fmt.Sprintf("%.2f MB", totalMB)
		}

		sTotalRequests := ac.FormatMoney(totalRequests)
		sTotalRequestsErr := ac.FormatMoney(totalRequestsErr)
		fmt.Fprintf(os.Stdout, "\rExitosas: %s | Errores: %s | Acierto: %.1f%% | Prom: %.2f KB | Transferido: %s ", sTotalRequests, sTotalRequestsErr, hitRate, sizeProm, transferStr)
	}
}

// update .syso
// $GOPATH/bin/rsrc -arch 386 -ico img/icon1.ico
// $GOPATH/bin/rsrc -arch amd64 -ico img/icon1.ico

// go build
