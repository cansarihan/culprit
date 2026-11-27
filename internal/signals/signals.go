// Package signals holds the evidence Culprit weighs when it asks whether a
// commit is a security fix. There are three families: what the message says,
// what the diff adds, and which files it touches. Each signal carries a weight;
// the scorer adds up the ones a commit trips.
package signals

import "sort"

// Family groups signals by where their evidence comes from.
type Family string

const (
	FamilyMessage Family = "message"
	FamilyDiff    Family = "diff"
	FamilyPath    Family = "path"
)

// Signal is one kind of evidence.
type Signal struct {
	ID          string `json:"id"`
	Family      Family `json:"family"`
	Label       string `json:"label"`
	Weight      int    `json:"weight"`
	Description string `json:"description"`
	// Announced marks a signal that says out loud the commit is about security,
	// such as a CVE identifier. A fix that trips one was not silent.
	Announced bool `json:"announced,omitempty"`
}

// Hit is a signal a commit tripped, with the text that tripped it.
type Hit struct {
	Signal   Signal `json:"signal"`
	Evidence string `json:"evidence"`
	File     string `json:"file,omitempty"`
}

var catalog []Signal

func define(s Signal) Signal {
	catalog = append(catalog, s)
	return s
}

// Catalog returns every signal, grouped by family and then by weight.
func Catalog() []Signal {
	out := append([]Signal(nil), catalog...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Family != out[j].Family {
			return out[i].Family < out[j].Family
		}
		return out[i].Weight > out[j].Weight
	})
	return out
}
