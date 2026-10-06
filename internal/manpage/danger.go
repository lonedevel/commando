package manpage

import (
	"regexp"
	"strings"
)

var (
	// The option's first sentence starts with a destructive action:
	// "remove directories and their contents", "Recursively remove all
	// files", "receiver deletes before xfer", "Attempt to remove the file".
	destroyRe = regexp.MustCompile(`(?i)^(?:attempt to |(\S+) )?(?:delete|deletes|remove|removes|overwrite|overwrites|unlink|unlinks|destroy|destroys|wipe|wipes|erase|erases|truncate|truncates|shred|shreds)\b`)
	// It turns off the confirmation the command would otherwise ask for.
	noPromptRe = regexp.MustCompile(`(?i)\b(?:never prompt|do not prompt|don't prompt|without prompting|without (?:asking|confirmation))\b`)
	// "force deletion of dirs even if not empty"
	forceRe = regexp.MustCompile(`(?i)\bforce (?:deletion|removal|overwrit)`)
	// Removing these isn't destructive: "remove any trailing slashes",
	// "Remove intermediate containers after a successful build".
	harmlessRe = regexp.MustCompile(`(?i)\b(?:slash|slashes|whitespace|prefix|suffix|duplicates?|components|leading|trailing|characters|empty lines|intermediate|temporary)\b`)
	// Nor is cleaning up after itself: "Automatically remove the container
	// … when it exits" (docker run --rm).
	selfCleanRe = regexp.MustCompile(`(?i)\bwhen (?:it|they|the \w+) (?:exits?|finish(?:es)?|stops?|completes?|terminates?)\b`)
)

// negations that can precede a verb without making it destructive.
var negations = map[string]bool{"never": true, "not": true, "no": true, "don't": true, "dont": true, "cannot": true, "won't": true}

// knownDanger flags well-known risky options whose manual wording the
// rules above miss. Keys are spec.Command values.
var knownDanger = map[string]map[string]string{
	"rm":           {"-r": "deletes directories and everything in them", "-R": "deletes directories and everything in them", "--recursive": "deletes directories and everything in them", "-f": "deletes without asking, even read-only files", "--force": "deletes without asking, even read-only files"},
	"rsync":        {"--del": "deletes files on the destination that aren't in the source"},
	"find":         {"-delete": "deletes every file that matches"},
	"git push":     {"-f": "can overwrite commits on the remote", "--force": "can overwrite commits on the remote", "--force-with-lease": "can overwrite commits on the remote", "--delete": "deletes branches or tags on the remote", "--mirror": "can delete and overwrite branches on the remote", "--prune": "deletes remote branches that don't exist locally"},
	"git reset":    {"--hard": "discards uncommitted changes in your working tree"},
	"git clean":    {"-f": "deletes untracked files", "--force": "deletes untracked files", "-x": "also deletes ignored files", "-X": "deletes ignored files"},
	"git checkout": {"-f": "discards local changes", "--force": "discards local changes"},
	"git branch":   {"-D": "deletes a branch even if it isn't merged"},
	"shred":        {"-u": "deletes the file after overwriting it", "--remove": "deletes the file after overwriting it"},
}

// Commands that destroy data by design: only knownDanger entries are
// flagged, since every option's description talks about overwriting.
var tableOnly = map[string]bool{"shred": true}

// ApplyDanger marks options that delete or overwrite data, or that skip
// confirmation, with a short reason the form shows as a warning.
func ApplyDanger(s *Spec) {
	known := knownDanger[s.Command]
	for i := range s.Options {
		o := &s.Options[i]
		for _, n := range o.Names {
			if r, ok := known[n]; ok {
				o.Danger = r
				break
			}
		}
		if o.Danger == "" && !tableOnly[s.Command] {
			o.Danger = dangerFromText(o.Desc)
		}
	}
}

func dangerFromText(desc string) string {
	first := firstParas(desc, 1)
	if sents := sentences(first); len(sents) > 0 {
		first = sents[0]
	}
	first = strings.TrimSpace(first)
	if first == "" {
		return ""
	}
	switch {
	case noPromptRe.MatchString(first):
		return "skips the confirmation prompt"
	case forceRe.MatchString(first):
		return "forces deleting or overwriting"
	}
	m := destroyRe.FindStringSubmatch(first)
	if m == nil || harmlessRe.MatchString(first) || selfCleanRe.MatchString(first) || negations[strings.ToLower(m[1])] ||
		strings.Contains(strings.ToLower(first), "(default)") {
		return ""
	}
	return "can delete or overwrite data"
}
