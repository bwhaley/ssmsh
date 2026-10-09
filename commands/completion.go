package commands

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/mattn/go-shellwords"
	"github.com/reeflective/readline"
)

type completionKey struct{ profile, region, prefix string }
type completionEntry struct {
	names   []string
	err     error
	expires time.Time
}

var completionCache = make(map[completionKey]completionEntry)

func clearCompletionCache() { completionCache = make(map[completionKey]completionEntry) }

// Complete is the interactive shell's Tab callback. It never reads values.
func Complete(line []rune, cursor int) readline.Completions {
	names, err := completionCandidates(line, cursor)
	if err != nil {
		return readline.CompleteMessage("Completion unavailable: %v", err)
	}
	return readline.CompleteValues(names...).NoSpace('/')
}

func completionCandidates(line []rune, cursor int) ([]string, error) {
	// Do not replace text after the cursor, or guess how to close quoted tokens.
	if cursor < 0 || cursor > len(line) || (cursor < len(line) && !unicode.IsSpace(line[cursor])) {
		return nil, nil
	}
	start := cursor
	for start > 0 && !unicode.IsSpace(line[start-1]) {
		start--
	}
	word := string(line[start:cursor])
	if strings.ContainsAny(word, "\"'\\") {
		return nil, nil
	}
	args, err := shellwords.Parse(string(line[:start]))
	if err != nil {
		return nil, nil
	}
	if len(args) == 0 || (len(args) == 1 && args[0] == "help") {
		names := []string{}
		for name := range handlers {
			if strings.HasPrefix(name, word) {
				names = append(names, name)
			}
		}
		if len(args) == 0 && strings.HasPrefix("help", word) {
			names = append(names, "help")
		}
		sort.Strings(names)
		return names, nil
	}
	cmd := args[0]
	positional := 0
	for _, arg := range args[1:] {
		if !strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "key=") {
			positional++
		}
	}
	option := ""
	switch cmd {
	case "ls", "get", "rm":
	case "cd", "history":
		if positional > 0 {
			return nil, nil
		}
	case "cp", "mv":
		if positional >= 2 {
			return nil, nil
		}
	case "put":
		if !strings.HasPrefix(word, "name=") {
			return nil, nil
		}
		option, word = "name=", strings.TrimPrefix(word, "name=")
	default:
		return nil, nil
	}
	if strings.HasPrefix(word, "-") || strings.Contains(word, "=") {
		return nil, nil
	}
	region, regionPrefix := ps.Region, ""
	if cmd == "put" {
		// put uses a separate region= option, not region-qualified names.
		if strings.Contains(word, ":") {
			return nil, nil
		}
		allArgs, parseErr := shellwords.Parse(string(line))
		if parseErr != nil {
			return nil, nil
		}
		for _, arg := range allArgs[1:] {
			field, value, ok := strings.Cut(arg, "=")
			if ok && strings.EqualFold(strings.TrimSpace(field), "region") {
				if value == "" {
					return nil, nil
				}
				region = value
			}
		}
	}
	if i := strings.IndexByte(word, ':'); i >= 0 {
		if i == 0 || strings.Contains(word[i+1:], ":") {
			return nil, nil
		}
		region, regionPrefix, word = word[:i], word[:i+1], word[i+1:]
	}
	// Resolve only the parent: cleaning the entire prefix would lose a trailing
	// slash and accidentally match siblings (e.g. /app/ versus /apple).
	i := strings.LastIndexByte(word, '/')
	parent, base := word[:i+1], word[i+1:]
	absoluteParent := strings.TrimSuffix(ps.Resolve(parent), "/") + "/"
	prefix := absoluteParent + base
	key := completionKey{ps.Profile, region, prefix}
	entry, ok := completionCache[key]
	if !ok || !time.Now().Before(entry.expires) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		names, lookupErr := ps.CompletionNames(ctx, prefix, region)
		cancel()
		ttl := 30 * time.Second
		if lookupErr != nil {
			ttl = time.Second
		}
		entry = completionEntry{names, lookupErr, time.Now().Add(ttl)}
		if len(completionCache) >= 64 {
			clearCompletionCache()
		}
		completionCache[key] = entry
	}
	if entry.err != nil {
		return nil, entry.err
	}
	seen := make(map[string]bool)
	for _, name := range entry.names {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		tail := strings.TrimPrefix(name, absoluteParent)
		directory := false
		if slash := strings.IndexByte(tail, '/'); slash >= 0 {
			tail, directory = tail[:slash+1], true
		}
		if cmd == "cd" && !directory {
			continue
		}
		seen[option+regionPrefix+parent+tail] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
