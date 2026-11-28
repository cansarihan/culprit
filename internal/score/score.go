// Package score turns the signals a commit trips into a verdict: how likely it
// is to be a security fix, and whether that fix was announced or slipped in
// quietly. A quiet fix is the interesting one — it marks a vulnerability that
// shipped and was patched without a word.
package score

import (
	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/patch"
	"github.com/cansarihan/culprit/internal/signals"
)

const (
	BandLikely   = "likely"
	BandPossible = "possible"
	BandUnlikely = "unlikely"

	// targetedChurn is the size under which a change counts as surgical.
	targetedChurn = 40
	targetedBonus = 8
)

// Assessment is the verdict on one commit.
type Assessment struct {
	Commit   gitlog.Commit `json:"commit"`
	Files    int           `json:"files"`
	Churn    int           `json:"churn"`
	Score    int           `json:"score"`
	Band     string        `json:"band"`
	Silent   bool          `json:"silent"`
	Targeted bool          `json:"targeted"`
	Hits     []signals.Hit `json:"hits"`
	Paths    []string      `json:"paths"`
}

// Assess scores a commit from its message and diff.
func Assess(c gitlog.Commit, files []patch.FileDiff) Assessment {
	a := Assessment{Commit: c, Files: len(files)}
	for _, f := range files {
		a.Churn += f.Churn()
		a.Paths = append(a.Paths, f.Path)
	}

	message := signals.MessageHits(c.Subject, c.Body)
	diff := signals.DiffHits(files)
	path := signals.PathHits(files)

	a.Hits = append(a.Hits, message...)
	a.Hits = append(a.Hits, diff...)
	substantive := len(message)+len(diff) > 0
	// A sensitive path on its own proves nothing; it only sharpens a commit
	// that already carries message or diff evidence.
	if substantive {
		a.Hits = append(a.Hits, path...)
	}

	announced := false
	for _, h := range a.Hits {
		a.Score += h.Signal.Weight
		if h.Signal.Announced {
			announced = true
		}
	}
	if substantive && a.Churn > 0 && a.Churn <= targetedChurn {
		a.Targeted = true
		a.Score += targetedBonus
	}
	if a.Score > 100 {
		a.Score = 100
	}
	a.Band = band(a.Score)
	a.Silent = !announced && a.Band != BandUnlikely
	return a
}

func band(score int) string {
	switch {
	case score >= 60:
		return BandLikely
	case score >= 30:
		return BandPossible
	default:
		return BandUnlikely
	}
}
