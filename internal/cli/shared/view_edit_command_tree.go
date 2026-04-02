package shared

import (
	"context"
	"errors"
	"flag"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type commandTextReplacement struct {
	old string
	new string
}

type textRewrittenCommandError struct {
	message string
	err     error
}

func (e *textRewrittenCommandError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *textRewrittenCommandError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// NormalizeViewEditCommandTree rewrites canonical leaf verbs so user-facing read
// commands prefer `view` over `get`, and a small allowlist of update-only
// commands prefer `edit` over `set`.
func NormalizeViewEditCommandTree(root *ffcli.Command, editPaths map[string]struct{}) *ffcli.Command {
	if root == nil || isDeprecatedCompatibilityAliasCommand(root) {
		return root
	}

	rootName := strings.TrimSpace(root.Name)
	if rootName == "" {
		return root
	}

	replacements := make([]commandTextReplacement, 0)
	changed := false

	var walk func(parent, current *ffcli.Command, path string)
	walk = func(parent, current *ffcli.Command, path string) {
		if current == nil || isDeprecatedCompatibilityAliasCommand(current) {
			return
		}

		for _, sub := range current.Subcommands {
			if sub == nil {
				continue
			}
			childName := strings.TrimSpace(sub.Name)
			if childName == "" {
				continue
			}
			walk(current, sub, strings.TrimSpace(path+" "+childName))
		}

		if len(current.Subcommands) > 0 {
			return
		}

		newName := ""
		switch strings.TrimSpace(current.Name) {
		case "get":
			newName = "view"
		case "set":
			if _, ok := editPaths[path]; ok {
				newName = "edit"
			}
		}
		if newName == "" || parent == nil {
			return
		}

		pathReplacements := renameLeafVerb(current, path, newName)
		if len(pathReplacements) == 0 {
			return
		}

		replacements = append(replacements, pathReplacements...)
		changed = true
	}

	walk(nil, root, "asc "+rootName)
	if !changed {
		return root
	}

	sortCommandTextReplacements(replacements)
	rewriteCommandStrings(root, replacements)
	rewriteCommandErrors(root, replacements)
	wrapUsageFuncsToHideDeprecatedAliases(root)
	return root
}

func renameLeafVerb(cmd *ffcli.Command, oldPath, newName string) []commandTextReplacement {
	if cmd == nil {
		return nil
	}

	oldName := strings.TrimSpace(cmd.Name)
	if oldName == "" || oldName == newName {
		return nil
	}

	newPath := replaceLastPathSegment(oldPath, newName)
	if newPath == oldPath {
		return nil
	}

	oldShortUsage := strings.TrimSpace(cmd.ShortUsage)
	oldCommandPath := strings.TrimSpace(strings.TrimPrefix(oldPath, "asc "))
	newCommandPath := strings.TrimSpace(strings.TrimPrefix(newPath, "asc "))

	cmd.Name = newName
	renameFlagSetLastToken(cmd.FlagSet, oldName, newName)
	rewriteLeadingVerbDescriptions(cmd, oldName, newName)

	shortUsage := oldShortUsage
	if shortUsage == "" {
		shortUsage = oldPath
	}

	return []commandTextReplacement{
		{old: oldPath, new: newPath},
		{old: oldCommandPath, new: newCommandPath},
	}
}

func replaceLastPathSegment(path, newName string) string {
	trimmed := strings.TrimSpace(path)
	lastSpace := strings.LastIndex(trimmed, " ")
	if lastSpace == -1 {
		return strings.TrimSpace(newName)
	}
	return strings.TrimSpace(trimmed[:lastSpace+1] + newName)
}

func renameFlagSetLastToken(fs *flag.FlagSet, oldName, newName string) {
	if fs == nil {
		return
	}

	output := fs.Output()
	usage := fs.Usage
	name := strings.TrimSpace(fs.Name())
	switch {
	case name == "":
		name = newName
	case name == oldName:
		name = newName
	case strings.HasSuffix(name, " "+oldName):
		name = strings.TrimSuffix(name, " "+oldName) + " " + newName
	default:
		name = newName
	}

	fs.Init(name, fs.ErrorHandling())
	if output != nil {
		fs.SetOutput(output)
	}
	if usage != nil {
		fs.Usage = usage
	}
}

func rewriteLeadingVerbDescriptions(cmd *ffcli.Command, oldName, newName string) {
	if cmd == nil {
		return
	}

	switch {
	case oldName == "get" && newName == "view":
		cmd.ShortHelp = rewriteLeadingVerbText(cmd.ShortHelp, []string{"Get ", "Fetch "}, "View ")
		cmd.LongHelp = rewriteLeadingVerbText(cmd.LongHelp, []string{"Get ", "Fetch "}, "View ")
	case oldName == "set" && newName == "edit":
		cmd.ShortHelp = rewriteLeadingVerbText(cmd.ShortHelp, []string{"Set "}, "Edit ")
		cmd.LongHelp = rewriteLeadingVerbText(cmd.LongHelp, []string{"Set "}, "Edit ")
	}
}

func rewriteLeadingVerbText(input string, oldPrefixes []string, newPrefix string) string {
	for _, prefix := range oldPrefixes {
		if strings.HasPrefix(input, prefix) {
			return newPrefix + strings.TrimPrefix(input, prefix)
		}
	}
	return input
}

func isDeprecatedCompatibilityAliasCommand(cmd *ffcli.Command) bool {
	if cmd == nil {
		return false
	}

	shortHelp := strings.ToLower(strings.TrimSpace(cmd.ShortHelp))
	longHelp := strings.ToLower(strings.TrimSpace(cmd.LongHelp))

	return strings.HasPrefix(shortHelp, "deprecated:") ||
		strings.HasPrefix(shortHelp, "compatibility alias") ||
		strings.HasPrefix(longHelp, "deprecated compatibility alias") ||
		strings.HasPrefix(longHelp, "compatibility alias")
}

func wrapUsageFuncsToHideDeprecatedAliases(cmd *ffcli.Command) {
	if cmd == nil {
		return
	}

	cmd.UsageFunc = wrapUsageFuncToHideDeprecatedAliases(cmd.UsageFunc)
	for _, sub := range cmd.Subcommands {
		wrapUsageFuncsToHideDeprecatedAliases(sub)
	}
}

func wrapUsageFuncToHideDeprecatedAliases(base func(*ffcli.Command) string) func(*ffcli.Command) string {
	if base == nil {
		base = DefaultUsageFunc
	}

	return func(cmd *ffcli.Command) string {
		if cmd == nil {
			return ""
		}

		clone := *cmd
		if len(cmd.Subcommands) > 0 {
			visible := make([]*ffcli.Command, 0, len(cmd.Subcommands))
			for _, sub := range cmd.Subcommands {
				if sub == nil || isDeprecatedCompatibilityAliasCommand(sub) {
					continue
				}
				visible = append(visible, sub)
			}
			clone.Subcommands = visible
		}

		return base(&clone)
	}
}

func sortCommandTextReplacements(replacements []commandTextReplacement) {
	sort.SliceStable(replacements, func(i, j int) bool {
		return len(replacements[i].old) > len(replacements[j].old)
	})
}

func rewriteCommandStrings(cmd *ffcli.Command, replacements []commandTextReplacement) {
	if cmd == nil {
		return
	}

	if cmd.ShortUsage != "" {
		cmd.ShortUsage = applyCommandTextReplacements(cmd.ShortUsage, replacements)
	}
	if cmd.ShortHelp != "" {
		cmd.ShortHelp = applyCommandTextReplacements(cmd.ShortHelp, replacements)
	}
	if cmd.LongHelp != "" {
		cmd.LongHelp = applyCommandTextReplacements(cmd.LongHelp, replacements)
	}
	if cmd.FlagSet != nil {
		cmd.FlagSet.VisitAll(func(f *flag.Flag) {
			f.Usage = applyCommandTextReplacements(f.Usage, replacements)
		})
	}

	for _, sub := range cmd.Subcommands {
		rewriteCommandStrings(sub, replacements)
	}
}

func rewriteCommandErrors(cmd *ffcli.Command, replacements []commandTextReplacement) {
	if cmd == nil {
		return
	}

	if cmd.Exec != nil {
		originalExec := cmd.Exec
		cmd.Exec = func(ctx context.Context, args []string) error {
			err := originalExec(ctx, args)
			if err == nil || errors.Is(err, flag.ErrHelp) {
				return err
			}

			rewritten := applyCommandTextReplacements(err.Error(), replacements)
			if rewritten == err.Error() {
				return err
			}

			return &textRewrittenCommandError{
				message: rewritten,
				err:     err,
			}
		}
	}

	for _, sub := range cmd.Subcommands {
		rewriteCommandErrors(sub, replacements)
	}
}

func applyCommandTextReplacements(input string, replacements []commandTextReplacement) string {
	output := input
	for _, replacement := range replacements {
		output = strings.ReplaceAll(output, replacement.old, replacement.new)
	}
	return output
}
