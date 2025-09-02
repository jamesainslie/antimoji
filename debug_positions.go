package main

import "fmt"

func main() {
	s := "Unicode 😀, emoticon :), custom :smile:"
	fmt.Printf("String: %s\n", s)
	fmt.Printf("Length: %d bytes\n", len(s))
	fmt.Println()

	bytePos := 0
	for i, r := range s {
		fmt.Printf("Rune %2d: %c (byte pos %2d-%2d)\n", i, r, bytePos, bytePos+len(string(r)))
		bytePos += len(string(r))
	}

	fmt.Println()
	fmt.Printf("Position of ':)' should be around: %d\n", findSubstring(s, ":)"))
	fmt.Printf("Position of ':smile:' should be around: %d\n", findSubstring(s, ":smile:"))
}

func findSubstring(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
