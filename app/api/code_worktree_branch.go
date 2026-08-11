package api

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const codeWorktreeBranchAliasMaxRunes = 48

func codeSessionWorktreeBranch(sessionID uint, repositoryAlias string) string {
	base := fmt.Sprintf("gopanel/code-%d", sessionID)
	alias := codeWorktreeBranchAlias(repositoryAlias)
	if alias == "" {
		return base
	}
	return base + "-" + alias
}

func codeWorktreeBranchAlias(alias string) string {
	var result strings.Builder
	lastSeparator := false
	for _, character := range strings.TrimSpace(alias) {
		allowed := unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' || character == '_'
		if allowed {
			result.WriteRune(character)
			lastSeparator = false
			continue
		}
		if !lastSeparator {
			result.WriteByte('-')
			lastSeparator = true
		}
	}
	normalized := strings.Trim(result.String(), "-")
	if utf8.RuneCountInString(normalized) <= codeWorktreeBranchAliasMaxRunes {
		return normalized
	}
	return string([]rune(normalized)[:codeWorktreeBranchAliasMaxRunes])
}

func codeSessionRepositoryBranches(sessionID uint, sources []aiProjectWorkspaceSource) map[string]string {
	branches := make(map[string]string, len(sources))
	usedAliases := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		baseAlias := codeWorktreeBranchAlias(source.LinkName)
		if baseAlias == "" {
			baseAlias = "repository"
		}
		alias := baseAlias
		for suffix := 2; ; suffix++ {
			if _, exists := usedAliases[alias]; !exists {
				usedAliases[alias] = struct{}{}
				break
			}
			alias = fmt.Sprintf("%s-%d", baseAlias, suffix)
		}
		branches[source.Path] = codeSessionWorktreeBranch(sessionID, alias)
	}
	return branches
}
