package util

import "strings"

func ReplaceEwith2Dots(s string) string {
	s = strings.ReplaceAll(s, "Ё", "Е")
	return strings.ReplaceAll(s, "ё", "е")
}
