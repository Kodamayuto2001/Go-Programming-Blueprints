package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"

	// "strings"
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// const otherWord = "*"

// var transforms = []string{
// 	otherWord,
// 	otherWord,
// 	otherWord,
// 	otherWord,
// 	otherWord + "app",
// 	otherWord + "site",
// 	otherWord + "time",
// 	"get" + otherWord,
// 	"go" + otherWord,
// 	"lets " + otherWord,
// }

type Sprinkle struct {
	ID		int
	Prefix	string
	Suffix 	string
}

func main() {
	//	絶対パスで読み込むしかない
	err := godotenv.Load(fmt.Sprintf("/mnt/d/workspace/go/Go-Programming-Blueprints/Chapter4-domain-search-command-line-tools/src/env/%s.env", os.Getenv("GO_ENV")))
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("mysql", os.Getenv("DB_USER")+":"+os.Getenv("DB_PASSWORD")+"@/"+os.Getenv("DB_NAME"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rand.Seed(time.Now().UTC().UnixNano())
	s := bufio.NewScanner(os.Stdin)
	
	// for s.Scan() {
	// 	t := transforms[rand.Intn(len(transforms))]
	// 	fmt.Println(strings.Replace(t, otherWord, s.Text(), -1))
	// }

	// var idList []interface{}
	// err = db.QueryRow("select id from sprinkle").Scan(&idList)
	// fmt.Println(idList)
	// fmt.Println(len(idList))
	// // id := rand.Intn(len(idList))
	// id := 5

	// var (
	// 	prefix string
	// 	suffix string
	// )
	// err = db.QueryRow("select prefix, suffix from sprinkle where id = ?", id).Scan(&prefix, &suffix)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(prefix)
	// fmt.Println(suffix)
	// fmt.Println(s)

	// rows, err := db.Query("select * from sprinkle")
	// if err != nil {
	// 	panic(err.Error())
	// }
	// defer rows.Close()

	// for rows.Next() {
	// 	var sprinkle Sprinkle
	// 	err := rows.Scan(&sprinkle.ID, &sprinkle.Prefix, &sprinkle.Suffix)
	// 	if err != nil {
	// 		panic(err.Error())
	// 	}
	// 	fmt.Println(sprinkle.ID, sprinkle.Prefix, sprinkle.Suffix)
	// }

	// fmt.Println("ok")
	// fmt.Println(s)

	rows, err := db.Query("select count(*) from sprinkle")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var count int 

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			log.Fatal(err)
		}
	}

	// fmt.Printf("Number of rows are %d\n", count)
	// fmt.Println(s)

	id := rand.Intn(count)

	var sprinkle Sprinkle
	err = db.QueryRow("select prefix, suffix from sprinkle where id = ?", id).Scan(&sprinkle.Prefix, &sprinkle.Suffix)

	if err != nil {
		log.Fatal(err)
	}

	for s.Scan() {
		fmt.Println("\n"+sprinkle.Prefix+s.Text()+sprinkle.Suffix+"\n")
	}
}
