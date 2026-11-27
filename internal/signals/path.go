package signals

import (
	"regexp"

	"github.com/cansarihan/culprit/internal/patch"
)

type pathRule struct {
	signal Signal
	re     *regexp.Regexp
}

var pathRules = []pathRule{
	{define(Signal{ID: "path-auth", Family: FamilyPath, Label: "auth or session code", Weight: 14,
		Description: "touches authentication, sessions, tokens or permissions"}),
		regexp.MustCompile(`(?i)(auth|login|session|token|jwt|oauth|permission|acl|rbac|password)`)},
	{define(Signal{ID: "path-crypto", Family: FamilyPath, Label: "crypto code", Weight: 14,
		Description: "touches cryptography, keys or certificates"}),
		regexp.MustCompile(`(?i)(crypto|cipher|tls|cert|keys?\b|sign|hmac|hash)`)},
	{define(Signal{ID: "path-parser", Family: FamilyPath, Label: "parser or decoder", Weight: 12,
		Description: "touches code that parses or decodes untrusted bytes"}),
		regexp.MustCompile(`(?i)(pars|decod|unmarshal|lexer|scanner|reader|codec|archive|zip|tar)`)},
	{define(Signal{ID: "path-boundary", Family: FamilyPath, Label: "request boundary", Weight: 8,
		Description: "touches HTTP handlers, uploads, SQL or templates"}),
		regexp.MustCompile(`(?i)(handler|upload|http|server|sql|query|template|render)`)},
}

// PathHits returns the path signals a commit's touched files trip.
func PathHits(files []patch.FileDiff) []Hit {
	var hits []Hit
	for _, rule := range pathRules {
		for _, f := range files {
			if isTestFile(f.Path) {
				continue
			}
			if rule.re.MatchString(f.Path) {
				hits = append(hits, Hit{Signal: rule.signal, Evidence: f.Path, File: f.Path})
				break
			}
		}
	}
	return hits
}
