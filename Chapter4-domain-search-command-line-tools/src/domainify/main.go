package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
	"unicode"
	"flag"
)

var tlds = []string{"com", "net"}

const allowedChars = "abcdefghijklmnopqrstuvwxyz0123456789_-"

func main() {
	flag.Parse()

	// fmt.Println(flag.Args())
	// fmt.Printf("%T\n",flag.Args())
	// fmt.Println(len(flag.Args()))
	if len(flag.Args()) > 0 {
		tlds = flag.Args()
	}

	rand.Seed(time.Now().UTC().UnixNano())
	s := bufio.NewScanner(os.Stdin)

	for s.Scan() {
		text := strings.ToLower(s.Text())
		var newText []rune
		for _, r := range text {
			if unicode.IsSpace(r) {
				r = '-'
			}
			if !strings.ContainsRune(allowedChars, r) {
				continue
			}
			newText = append(newText, r)
		}
		if string(newText) != "" {
			fmt.Println(string(newText) + "." + tlds[rand.Intn(len(tlds))])
		}
	}
}
