package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"html"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

//go:embed config/it_IT.dic
var embeddedFS embed.FS

var (
	digitsRe       = regexp.MustCompile(`^\d{1,4}$`)
	accentReplacer = strings.NewReplacer(
		"à", "a", "á", "a", "è", "e", "é", "e", "ì", "i", "í", "i", "ò", "o", "ó", "o", "ù", "u", "ú", "u",
		"À", "a", "Á", "a", "È", "e", "É", "e", "Ì", "i", "Í", "i", "Ò", "o", "Ó", "o", "Ù", "u", "Ú", "u",
	)
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	limit := fs.Int("limit", 0, "massimo numero di parole da stampare per numero (0 = nessun limite)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *limit < 0 {
		fmt.Fprintln(os.Stderr, "errore: -limit deve essere >= 0")
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "uso: go run . [-limit N] <numero 001-1000 | intervallo 001-1000>")
		os.Exit(2)
	}

	targets, err := parseInputSpec(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(2)
	}

	raw, err := embeddedFS.ReadFile("config/it_IT.dic")
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore lettura dizionario embedded:", err)
		os.Exit(1)
	}

	lemmas, err := parseHunspellLemmas(string(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore parsing dizionario:", err)
		os.Exit(1)
	}

	for _, targetDigits := range targets {
		matches := make([]string, 0, 128)
		seen := make(map[string]struct{}, 1024)

		for _, lemma := range lemmas {
			word := normalizeWord(lemma)
			if word == "" {
				continue
			}
			if _, ok := seen[word]; ok {
				continue
			}
			if matchesTarget(word, targetDigits) {
				seen[word] = struct{}{}
				matches = append(matches, word)
			}
		}

		sort.Strings(matches)
		if *limit > 0 && len(matches) > *limit {
			matches = matches[:*limit]
		}

		if len(targets) == 1 {
			for _, w := range matches {
				fmt.Println(w)
			}
			continue
		}

		if len(matches) == 0 {
			continue
		}
		fmt.Printf("%s\t\n", targetDigits)
		for _, w := range matches {
			fmt.Println(w)
		}
	}
}

func normalizeInputNumber(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !digitsRe.MatchString(s) {
		return "", errors.New("il parametro deve contenere solo cifre")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return "", errors.New("numero non valido")
	}
	if n < 1 || n > 1000 {
		return "", errors.New("il numero deve essere tra 001 e 1000")
	}
	if n == 1000 {
		return "1000", nil
	}
	if len(s) < 3 {
		s = fmt.Sprintf("%03d", n)
	}
	if len(s) > 3 {
		return "", errors.New("per valori inferiori a 1000 usare al massimo 3 cifre")
	}
	return s, nil
}

func parseInputSpec(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("parametro vuoto")
	}

	if !strings.Contains(s, "-") {
		num, err := normalizeInputNumber(s)
		if err != nil {
			return nil, err
		}
		return []string{num}, nil
	}

	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return nil, errors.New("range non valido, usa formato tipo 001-003")
	}
	startStr, err := normalizeInputNumber(parts[0])
	if err != nil {
		return nil, fmt.Errorf("inizio range non valido: %w", err)
	}
	endStr, err := normalizeInputNumber(parts[1])
	if err != nil {
		return nil, fmt.Errorf("fine range non valida: %w", err)
	}

	start, _ := strconv.Atoi(startStr)
	end, _ := strconv.Atoi(endStr)
	if start > end {
		return nil, errors.New("range non valido: l'inizio deve essere <= della fine")
	}

	targets := make([]string, 0, end-start+1)
	for n := start; n <= end; n++ {
		targets = append(targets, formatNumber(n))
	}
	return targets, nil
}

func formatNumber(n int) string {
	if n == 1000 {
		return "1000"
	}
	return fmt.Sprintf("%03d", n)
}

func parseHunspellLemmas(raw string) ([]string, error) {
	data := strings.TrimSpace(raw)
	if strings.HasPrefix(data, "<!DOCTYPE html>") {
		start := strings.Index(data, "<td class='lines'><pre><code>")
		end := strings.Index(data, "</code></pre></td>")
		if start < 0 || end < 0 || end <= start {
			return nil, errors.New("html del dizionario non contiene il blocco con le righe")
		}
		start += len("<td class='lines'><pre><code>")
		data = html.UnescapeString(data[start:end])
	}

	lines := strings.Split(data, "\n")
	if len(lines) == 0 {
		return nil, errors.New("dizionario vuoto")
	}

	lemmas := make([]string, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i == 0 && isNumber(line) {
			continue
		}

		if slash := strings.IndexByte(line, '/'); slash >= 0 {
			line = line[:slash]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			lemmas = append(lemmas, line)
		}
	}

	if len(lemmas) == 0 {
		return nil, errors.New("nessun lemma estratto")
	}
	return lemmas, nil
}

func isNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func normalizeWord(s string) string {
	s = strings.TrimSpace(s)
	s = accentReplacer.Replace(s)
	s = strings.ToLower(s)

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func matchesTarget(word string, targetDigits string) bool {
	wordDigits, ok := wordToDigits(word)
	if !ok {
		return false
	}
	return wordDigits == targetDigits
}

func wordToDigits(word string) (string, bool) {
	var b strings.Builder
	b.Grow(len(word))
	lastDigit := byte(0)
	hasLastDigit := false
	hadVowelSinceLastDigit := false

	for i := 0; i < len(word); i++ {
		ch := word[i]
		next := byte(0)
		next2 := byte(0)
		if i+1 < len(word) {
			next = word[i+1]
		}
		if i+2 < len(word) {
			next2 = word[i+2]
		}

		// "sc" davanti a e/i ha suono dolce unico.
		if ch == 's' && next == 'c' && isFrontVowel(next2) {
			appendDigit(&b, '6', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
			i++
			continue
		}

		switch ch {
		case 'a', 'e', 'i', 'o', 'u', 'h', 'y':
			if hasLastDigit {
				hadVowelSinceLastDigit = true
			}
			continue
		case 's', 'z', 'x':
			appendDigit(&b, '0', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 't', 'd':
			appendDigit(&b, '1', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 'n':
			appendDigit(&b, '2', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 'm':
			appendDigit(&b, '3', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 'r':
			appendDigit(&b, '4', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 'l':
			appendDigit(&b, '5', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 'j':
			appendDigit(&b, '6', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 'c':
			if next == 'h' {
				appendDigit(&b, '7', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
				i++
			} else if isFrontVowel(next) {
				appendDigit(&b, '6', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
			} else {
				appendDigit(&b, '7', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
			}
		case 'g':
			if next == 'h' {
				appendDigit(&b, '7', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
				i++
			} else if isFrontVowel(next) {
				appendDigit(&b, '6', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
			} else {
				appendDigit(&b, '7', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
			}
		case 'k', 'q':
			appendDigit(&b, '7', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 'f', 'v':
			appendDigit(&b, '8', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		case 'p', 'b':
			appendDigit(&b, '9', &lastDigit, &hasLastDigit, &hadVowelSinceLastDigit)
		default:
			return "", false
		}
	}
	return b.String(), true
}

func appendDigit(b *strings.Builder, digit byte, lastDigit *byte, hasLastDigit *bool, hadVowelSinceLastDigit *bool) bool {
	// Consonanti uguali adiacenti (senza vocale in mezzo) valgono come una sola cifra.
	if *hasLastDigit && *lastDigit == digit && !*hadVowelSinceLastDigit {
		return false
	}
	b.WriteByte(digit)
	*lastDigit = digit
	*hasLastDigit = true
	*hadVowelSinceLastDigit = false
	return true
}

func isFrontVowel(ch byte) bool {
	return ch == 'e' || ch == 'i'
}
