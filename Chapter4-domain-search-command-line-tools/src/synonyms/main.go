package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"../thesaurus"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(fmt.Sprintf("/mnt/d/workspace/go/Go-Programming-Blueprints/Chapter4-domain-search-command-line-tools/src/env/%s.env", os.Getenv("GO_ENV")))
	if err != nil {
		log.Fatal(err)
	}

	apiKey := os.Getenv("BHT_APIKEY")

	thesaurus := &thesaurus.BigHuge{APIKey: apiKey}
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		word := s.Text()
		syns, err := thesaurus.Synonyms(word)
		if err != nil {
			log.Fatalf("%qの類語検索に失敗しました： %v\n", word, err)
		}
		if len(syns) == 0 {
			log.Fatalf("%qに類語はありませんでした\n")
		}
		for _, syn := range syns {
			fmt.Println(syn)
		}
	}
}
