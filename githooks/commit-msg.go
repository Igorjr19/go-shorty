package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: Commit message file path not provided.")
		os.Exit(1)
	}

	commitMsgFile := os.Args[1]

	content, err := os.ReadFile(commitMsgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler o arquivo de mensagem de commit: %v\n", err)
		os.Exit(1)
	}

	commitMsg := string(content)

	lines := strings.Split(commitMsg, "\n")
	var subjectLine string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			subjectLine = line
			break
		}
	}

	regexPattern := `^(feat|fix|docs|style|refactor|test|chore|build|ci|perf|revert)(\([a-zA-Z0-9_.-]+\))?(!)?:\s.*$`
	matched, err := regexp.MatchString(regexPattern, subjectLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing regex: %v\n", err)
		os.Exit(1)
	}

	if !matched {
		printErrorAndExit()
	}

	os.Exit(0)
}

func printErrorAndExit() {
	msg := `Error: The commit message does not follow the Conventional Commits format.`
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
