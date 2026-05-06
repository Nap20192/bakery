package bot

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var dateLineRe = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}$`)

type parsedLine struct {
	Name     string
	Quantity int
}

// isBulkOrder возвращает true, если текст выглядит как многострочная заявка.
func isBulkOrder(text string) bool {
	return strings.Contains(strings.TrimSpace(text), "\n")
}

// parseBulkOrder разбирает текст заявки формата:
//
//	Локация
//
//	КАТЕГОРИЯ
//	Продукт количество
//	Продукт (количество = 1 если не указано)
//
// Старый формат со строкой ДД.ММ.ГГГГ тоже поддерживается, но дата не обязательна.
func parseBulkOrder(text string) (location, date string, lines []parsedLine) {
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if location == "" {
			location = line
			continue
		}
		if date == "" && dateLineRe.MatchString(line) {
			date = line
			continue
		}
		if isCategoryHeader(line) {
			continue
		}
		name, qty := parseProductLine(line)
		if name != "" {
			lines = append(lines, parsedLine{Name: name, Quantity: qty})
		}
	}
	return
}

// isCategoryHeader — строка из заглавных букв (КОКРОКИ, ПИРОГИ СЫТНЫЕ/СЛАДКИЕ и т.п.)
func isCategoryHeader(line string) bool {
	hasLetter := false
	for _, r := range line {
		if unicode.IsLetter(r) {
			hasLetter = true
			if unicode.IsLower(r) {
				return false
			}
		}
	}
	return hasLetter
}

// parseProductLine выделяет название и количество из строки вида "Круассан с шоколадом 25".
// Если число в конце отсутствует — количество = 1.

func parseProductLine(line string) (name string, qty int) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", 0
	}
	last := fields[len(fields)-1]
	if n, err := strconv.Atoi(last); err == nil && n > 0 {
		return strings.Join(fields[:len(fields)-1], " "), n
	}
	return strings.Join(fields, " "), 1
}
