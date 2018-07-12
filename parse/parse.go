package parse

import (
	"fmt"
	"strconv"
	"time"

	"github.com/looyun/feedall/controllers"
	"github.com/looyun/feedall/models"
	"github.com/mmcdole/gofeed"
	"gopkg.in/mgo.v2/bson"
)

func Parse() {
	for {
		timer := time.NewTimer(60 * time.Second)
		fmt.Println("start parse!")
		feedlist := make([]*models.FeedList, 0)
		if !models.FindAll(models.FeedLists, nil, &feedlist) {
			fmt.Println(<-timer.C)
			continue
		} else {
			Finish := make(chan string)
			fb := gofeed.NewParser()
			for _, u := range feedlist {
				go func(u *models.FeedList) {
					origin_feed, err := fb.ParseURL(u.FeedLink)
					if err != nil {
						fmt.Println("Parse err: ", err)
						Finish <- u.FeedLink
					} else {
						data, err := bson.Marshal(origin_feed)
						if err != nil {
							fmt.Println(err)
						}
						var items struct {
							Items []*models.Item `bson:"items"`
						}
						err = bson.Unmarshal(data, &items)
						if err != nil {
							fmt.Println(err)
						}
						feed := models.Feed{}
						models.FindOne(models.Feeds,
							bson.M{"feedLink": u.FeedLink},
							&feed)
						for _, v := range items.Items {
							if v.Content == "" {
								if v.Extensions != nil && v.Extensions["content"] != nil {
									v.Content = v.Extensions["content"]["encoded"][0].Value
								} else {
									v.Content = v.Description
								}
							}
							v.Content = controllers.DecodeImg(v.Content, u.Link)
							if v.Published == "" {
								v.Published = v.Updated
							}
							publishedParsed := controllers.ParseDate(v.Published)
							v.PublishedParsed = strconv.FormatInt(publishedParsed.Unix(), 10)
							v.FeedID = feed.ID

							info, err := models.Upsert(models.Items,
								bson.M{"link": v.Link},
								v)
							if err != nil {
								fmt.Println(err)
							}
							if info == nil {
								continue
							}
							if info.Updated > 0 {
								fmt.Printf("updated %d\n", info.Updated)
							} else {
								fmt.Println("no new item")
							}
						}
						Finish <- u.FeedLink
					}
				}(u)
			}
			for _, _ = range feedlist {
				fmt.Println(<-Finish)
			}
		}
		fmt.Println("OK!")
		fmt.Println(<-timer.C)
	}
}
