package signals

import (
	"regexp"
	"strings"
)

type messageRule struct {
	signal Signal
	re     *regexp.Regexp
}

var messageRules = []messageRule{
	{define(Signal{ID: "msg-cve", Family: FamilyMessage, Label: "CVE or advisory id", Weight: 40, Announced: true,
		Description: "names a CVE or GHSA advisory; the fix was disclosed"}),
		regexp.MustCompile(`(?i)\b(CVE-\d{4}-\d{4,}|GHSA(-[a-z0-9]{4}){3})\b`)},
	{define(Signal{ID: "msg-security", Family: FamilyMessage, Label: "says security", Weight: 30, Announced: true,
		Description: "calls itself a security or vulnerability fix"}),
		regexp.MustCompile(`(?i)\b(security|vulnerab\w*|exploit\w*)\b`)},
	{define(Signal{ID: "msg-memory", Family: FamilyMessage, Label: "memory-safety words", Weight: 22,
		Description: "talks about overflows, out-of-bounds access or use-after-free"}),
		regexp.MustCompile(`(?i)\b(overflow|out[- ]of[- ]bounds|oob|underflow|use[- ]after[- ]free|double free|bounds)\b`)},
	{define(Signal{ID: "msg-input", Family: FamilyMessage, Label: "input hardening", Weight: 18,
		Description: "sanitizes, validates, escapes or rejects input"}),
		regexp.MustCompile(`(?i)\b(sanitiz\w*|validat\w*|escap\w*|reject\w*|harden\w*|malformed|untrusted)\b`)},
	{define(Signal{ID: "msg-crash", Family: FamilyMessage, Label: "crash or panic", Weight: 14,
		Description: "fixes a crash, panic or nil dereference"}),
		regexp.MustCompile(`(?i)\b(crash\w*|panic\w*|nil (pointer|deref\w*)|segfault)\b`)},
	{define(Signal{ID: "msg-resource", Family: FamilyMessage, Label: "resource limits", Weight: 12,
		Description: "adds a limit, timeout or guard against exhaustion"}),
		regexp.MustCompile(`(?i)\b(limit\w*|timeout|denial of service|dos|exhaust\w*|unbounded|infinite loop)\b`)},
	{define(Signal{ID: "msg-vague", Family: FamilyMessage, Label: "vague fix", Weight: 6,
		Description: "a terse 'fix', 'cleanup' or 'misc' message — the classic cover for a quiet patch"}),
		regexp.MustCompile(`(?i)^(fix|fixes|fixed|minor fix(es)?|cleanup|clean up|misc|small fix(es)?|tweak|update)\.?$`)},
}

// MessageHits returns the message signals a commit's subject and body trip.
func MessageHits(subject, body string) []Hit {
	text := subject + "\n" + body
	var hits []Hit
	for _, rule := range messageRules {
		target := text
		if rule.signal.ID == "msg-vague" {
			target = strings.TrimSpace(subject)
		}
		if m := rule.re.FindString(target); m != "" {
			hits = append(hits, Hit{Signal: rule.signal, Evidence: m})
		}
	}
	return hits
}
