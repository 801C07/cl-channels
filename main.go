package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/anaskhan96/soup"
)

// Item is the data we want from the response
type Item struct {
	RemoteID   int
	BatchID    int
	Title      string
	Price      int
	DatePosted string
	Seen       string
}

func main() {
	ch := make(chan []Item, 5)

	go func() {
		for x := 0; x < 3000; x += 120 {
			ch <- getItems(x)
		}
		close(ch)
	}()

	for {
		x, ok := <-ch
		if ok {
			err := putItemsInDB(x)
			check(err)
		} else {
			break
		}
	}

}

func putItemsInDB(items []Item) error {
	db, err := sql.Open("mysql", "root:yourpassword@tcp(localhost:3306)/catan")
	check(err)

	stmt, err := db.Prepare("insert ignore into cl VALUES(?,?,?,?,?,?)")
	check(err)

	for _, item := range items {
		_, err := stmt.Exec(item.RemoteID, item.BatchID, item.Title, item.Price, item.DatePosted, item.Seen)
		check(err)
	}

	return nil
}

func getItems(x int) []Item {
	resp, err := http.Get(fmt.Sprintf("https://boise.craigslist.org/search/sss?s=%d", x))
	check(err)

	body, err := ioutil.ReadAll(resp.Body)
	check(err)

	Items, _, err := ParsePage(string(body), 2)
	check(err)

	return Items
}

// ParsePage from the html
func ParsePage(html string, BatchID int) ([]Item, int, error) {
	var results []Item

	doc := soup.HTMLParse(html)

	NumResults, err := strconv.Atoi(doc.Find("span", "class", "totalcount").Text())
	check(err)

	Items := doc.Find("ul", "class", "rows").FindAll("li")

	for _, item := range Items {
		RemoteID, err := strconv.Atoi(item.Attrs()["data-pid"])
		check(err)

		Price := FindPrice(item)
		Title := item.Find("a", "class", "result-title").Text()
		DatePosted := item.Find("time", "class", "result-date").Attrs()["datetime"]
		Seen := time.Now().String()
		result := Item{RemoteID, BatchID, Title, Price, DatePosted, Seen}

		results = append(results, result)
	}

	return results, NumResults, nil
}

// FindPrice method to handle case where there is no price
func FindPrice(Item soup.Root) int {
	PriceText := Item.Find("span", "class", "result-price")

	if PriceText.Error == nil {
		Price, err1 := strconv.Atoi(strings.Trim(PriceText.Text(), "$"))
		check(err1)
		return Price
	}

	return 0
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
