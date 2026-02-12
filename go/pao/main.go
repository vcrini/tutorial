package main

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

//go:embed config/current_version/morph-it_048.txt
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
	morphPath := fs.String("morph", "", "percorso a lessico POS Morph-it (opzionale, override del file embedded)")
	concreteOnly := fs.Bool("concrete-only", true, "filtra nomi astratti e forme non concrete")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *limit < 0 {
		fmt.Fprintln(os.Stderr, "errore: -limit deve essere >= 0")
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "uso: go run . [-limit N] [-morph FILE] [-concrete-only=true|false] <numero 001-1000 | intervallo 001-1000>")
		os.Exit(2)
	}

	targets, err := parseInputSpec(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(2)
	}

	lemmas, err := loadMorphLemmas(*morphPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore lettura/parsing lessico Morph-it:", err)
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
			if *concreteOnly && !isLikelyConcreteNoun(word) {
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

func isLikelyConcreteNoun(word string) bool {
	if len(word) < 3 {
		return false
	}

	// Heuristics: scarta forme verbali/aggettivali e nomi astratti più comuni.
	blockedSuffixes := []string{
		"are", "ere", "ire",
		"arsi", "ersi", "irsi",
		"ando", "endo",
		"ato", "ata", "ati", "ate",
		"uto", "uta", "uti", "ute",
		"ito", "ita", "iti", "ite",
		"zione", "zioni", "mento", "menti", "ismo", "ismi", "ezza", "ezze", "anza", "anze", "enza", "enze",
		"mente",
		"ale", "ali", "ile", "ili", "iale", "iali", "oso", "osa", "osi", "ose",
		"ico", "ica", "ici", "iche", "ivo", "iva", "ivi", "ive",
	}
	for _, sfx := range blockedSuffixes {
		if strings.HasSuffix(word, sfx) {
			return false
		}
	}

	last := word[len(word)-1]
	switch last {
	case 'a', 'e', 'i', 'o':
		return true
	default:
		return false
	}
}

func loadMorphLemmas(path string) ([]string, error) {
	var r io.Reader
	if path == "" {
		raw, err := embeddedFS.ReadFile("config/current_version/morph-it_048.txt")
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(raw)
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}

	lemmasSet, err := parseMorphNouns(r)
	if err != nil {
		return nil, err
	}
	lemmas := make([]string, 0, len(lemmasSet))
	for lemma := range lemmasSet {
		lemmas = append(lemmas, lemma)
	}
	return lemmas, nil
}

func parseMorphNouns(r io.Reader) (map[string]struct{}, error) {
	nouns := make(map[string]struct{}, 64_000)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		lemma := normalizeWord(parts[1])
		if lemma == "" {
			continue
		}
		tag := strings.ToUpper(parts[2])
		if !looksLikeNounTag(tag) {
			continue
		}
		nouns[lemma] = struct{}{}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return nouns, nil
}

func looksLikeNounTag(tag string) bool {
	return strings.Contains(tag, "NOUN") ||
		strings.Contains(tag, "NOM") ||
		strings.Contains(tag, "SOST")
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
