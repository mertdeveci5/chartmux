package chartmux

var terminalAreaPatterns = [...]rune{'░', '▒', '▦', '▧', '▨', '▩'}

func terminalAreaPattern(index int) rune {
	return terminalAreaPatterns[index%len(terminalAreaPatterns)]
}
