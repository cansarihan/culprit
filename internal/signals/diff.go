package signals

import (
	"regexp"
	"strings"

	"github.com/cansarihan/culprit/internal/patch"
)

type diffRule struct {
	signal Signal
	match  func(f patch.FileDiff) (string, bool)
}

// addedMatching reports the first added line that matches re.
func addedMatching(re *regexp.Regexp) func(patch.FileDiff) (string, bool) {
	return func(f patch.FileDiff) (string, bool) {
		for _, line := range f.Added {
			if re.MatchString(line) {
				return strings.TrimSpace(line), true
			}
		}
		return "", false
	}
}

// swapped reports a commit that removed a line matching from and added one
// matching to — one API replaced by a safer one.
func swapped(from, to *regexp.Regexp) func(patch.FileDiff) (string, bool) {
	return func(f patch.FileDiff) (string, bool) {
		removed, added := "", ""
		for _, l := range f.Removed {
			if from.MatchString(l) {
				removed = strings.TrimSpace(l)
				break
			}
		}
		for _, l := range f.Added {
			if to.MatchString(l) {
				added = strings.TrimSpace(l)
				break
			}
		}
		if removed != "" && added != "" {
			return removed + "  →  " + added, true
		}
		return "", false
	}
}

var diffRules = []diffRule{
	{define(Signal{ID: "diff-bounds", Family: FamilyDiff, Label: "new bounds check", Weight: 26,
		Description: "adds a comparison against len() or cap() before an access"}),
		addedMatching(regexp.MustCompile(`if .*(len|cap)\(.*\)\s*[<>]=?|if .*[<>]=?\s*(len|cap)\(`))},
	{define(Signal{ID: "diff-traversal", Family: FamilyDiff, Label: "path traversal guard", Weight: 28,
		Description: "cleans a path or rejects '..' components"}),
		addedMatching(regexp.MustCompile(`filepath\.(Clean|Rel|IsLocal)\(|strings\.(Contains|HasPrefix)\([^)]*"\.\."|filepath\.IsAbs`))},
	{define(Signal{ID: "diff-consttime", Family: FamilyDiff, Label: "constant-time compare", Weight: 30,
		Description: "replaces an equality check on a secret with a constant-time comparison"}),
		addedMatching(regexp.MustCompile(`subtle\.ConstantTimeCompare|hmac\.Equal`))},
	{define(Signal{ID: "diff-limit", Family: FamilyDiff, Label: "input size limit", Weight: 22,
		Description: "caps how much input is read"}),
		addedMatching(regexp.MustCompile(`http\.MaxBytesReader|io\.LimitReader|io\.LimitedReader|MaxHeaderBytes|ReadLimit`))},
	{define(Signal{ID: "diff-escape", Family: FamilyDiff, Label: "output escaping", Weight: 24,
		Description: "escapes output for HTML, shell, SQL or regular expressions"}),
		addedMatching(regexp.MustCompile(`html\.EscapeString|template\.(HTML|JS)Escape|url\.QueryEscape|regexp\.QuoteMeta|shellescape|strconv\.Quote`))},
	{define(Signal{ID: "diff-timeout", Family: FamilyDiff, Label: "timeout or deadline", Weight: 14,
		Description: "adds a timeout, deadline or cancellable context"}),
		addedMatching(regexp.MustCompile(`context\.With(Timeout|Deadline)|Set(Read|Write)?Deadline|ReadHeaderTimeout|Timeout:\s`))},
	{define(Signal{ID: "diff-nilcheck", Family: FamilyDiff, Label: "new nil check", Weight: 10,
		Description: "guards a value against nil before use"}),
		addedMatching(regexp.MustCompile(`if [\w.\[\]]+ (==|!=) nil\b`))},
	{define(Signal{ID: "diff-lock", Family: FamilyDiff, Label: "new locking", Weight: 12,
		Description: "adds a mutex or atomic operation — the shape of a race fix"}),
		addedMatching(regexp.MustCompile(`\.(R?Lock)\(\)|sync\.(Mutex|RWMutex)|atomic\.\w+\(`))},
	{define(Signal{ID: "diff-cryptorand", Family: FamilyDiff, Label: "math/rand → crypto/rand", Weight: 30,
		Description: "swaps a predictable random source for a cryptographic one"}),
		swapped(regexp.MustCompile(`"math/rand"|rand\.(Intn|Int63|Read)\(`), regexp.MustCompile(`"crypto/rand"`))},
	{define(Signal{ID: "diff-sqlparam", Family: FamilyDiff, Label: "SQL placeholders", Weight: 32,
		Description: "replaces a formatted SQL string with a parameterized query"}),
		swapped(regexp.MustCompile(`fmt\.Sprintf\(\s*"(?i:select|insert|update|delete)`), regexp.MustCompile(`(Query|Exec|QueryRow)(Context)?\([^)]*(\$\d|\?)`))},
}

// DiffHits returns the diff signals a commit's changes trip, at most once per
// signal, recording the first file and line that tripped it.
func DiffHits(files []patch.FileDiff) []Hit {
	var hits []Hit
	for _, rule := range diffRules {
		for _, f := range files {
			if isTestFile(f.Path) {
				continue
			}
			if evidence, ok := rule.match(f); ok {
				hits = append(hits, Hit{Signal: rule.signal, Evidence: evidence, File: f.Path})
				break
			}
		}
	}
	return hits
}

func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/testdata/")
}
