package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/robfig/cron"
	"github.com/xujiajun/nutsdb"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var conf *IYaml //

func main() {
	fmt.Println("监听内容")

	ch := make(chan int, 1)
	conf = new(IYaml)
	yamlFile, err := ioutil.ReadFile("conf.yaml")


	log.Println("yamlFile:", yamlFile)
	if err != nil {
		log.Printf("yamlFile.Get err #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, conf)

	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	log.Println("conf", conf)

	c := cron.New()
	c.AddFunc("0 * * * * *", func() {
		for _,book := range conf.Books {
			getStoryLast(book.Name, book.Url, book.Selector)
		}

	})
	c.Start()
	<-ch
}

type IYaml struct {
	Books []Book
	Im string
}

type Book struct {
	Name  string
	Url string
	Selector string
}

func getStoryLast(bookName string, url string, selector string) {
	res, err := http.Get(url)

	opt := nutsdb.DefaultOptions
	opt.Dir = "./nutsdb"
	db, err := nutsdb.Open(opt)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}
	var lastTitle string

	db.View(func(tx *nutsdb.Tx) error {
		text, er := tx.Get("books", []byte(bookName))
		if (er == nil) {
			lastTitle = string(text.Value)
		}
		return nil
	});
	if (err == nil) {
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			panic(err)
		}
		lastNode := doc.Find(selector)
		currentTitle := lastNode.Find("a").Text()
		log.Println("《" +bookName+"》小说最新章节名为 =>", currentTitle)
		if (lastTitle != currentTitle) {
			db.Update(func(tx *nutsdb.Tx) error {
				tx.Put("books", []byte(bookName), []byte(currentTitle), nutsdb.Persistent)
				return nil
			})
			log.Println("章节有更新:",bookName+"更新了,新的一个章节为:"+currentTitle,"", url)
			sendMsg("《" +bookName+"》更新了,新的一个章节为:"+currentTitle,"", url)
		}

	} else {
		log.Println("error =>", err)
	}
}

func sendMsg(title string, context string, link string) {
	url := conf.Im
	data := strings.Replace(`{
					"msgtype": "link",
						"link": {
						"text": "$text", 
						"title": "$title",
						"picUrl": "",
						"messageUrl": "$messageUrl"
					}
				}`, "$title", title, -1)
	data = strings.Replace(data, "$text", title, -1)
	data = strings.Replace(data, "$messageUrl", link, -1)

	resp,err:= http.Post(url, "application/json", strings.NewReader(data))
	if(err == nil){
		ds, _ := ioutil.ReadAll(resp.Body)
		log.Println(string(ds))

	}
}
