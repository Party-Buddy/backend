package util

func SymbolsCount(s string) int {
	return len([]rune(s))
}

func MaxLengthPChecker(c int) func(v *string) bool {
	return func(v *string) bool {
		if v == nil {
			return false
		}
		return SymbolsCount(*v) <= c
	}
}

func MaxLengthChecker(c int) func(v string) bool {
	return func(v string) bool {
		return SymbolsCount(v) <= c
	}
}
