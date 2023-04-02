package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/turnage/graw/reddit"
)

func getAllComments(bot reddit.Bot, post *reddit.Post) {
	var queue []*reddit.Comment
	queue = append(queue, post.Replies...)
	for len(queue) != 0 {
		fmt.Println(len(queue))
		r := queue[0]
		queue = queue[1:]
		if r.More != nil {
			morechildren := map[string]string{
				"link_id":  "t3_" + post.ID,
				"children": r.More.Children[0],
			}
			harvest, err := bot.ListingWithParams("/api/morechildren.json", morechildren)
			if err != nil {
				panic(err)
			}
			r.Replies = append(r.Replies, harvest.Comments...)
		}
		queue = append(queue, r.Replies...)
	}
}

func writeComments(bot reddit.Bot, permalink string) {
	fmt.Println(permalink)
	post, _ := bot.Thread(permalink)
	getAllComments(bot, post)
	thread_json, _ := json.Marshal(post)

	f, err := os.Create("threads/" + post.ID + ".json")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	nbytes, err := f.Write(thread_json)
	if err != nil {
		panic(err)
	}
	fmt.Printf("wrote %d bytes\n", nbytes)

}

func main() {
	bot, err := reddit.NewBotFromAgentFile("lurker-bot.agent", 0)
	if err != nil {
		fmt.Println("Failed to create bot handle: ", err)
		return
	}

	harvest, err := bot.Listing("/r/anime", "")
	if err != nil {
		fmt.Println("Failed to fetch /r/anime: ", err)
		return
	}

	re := regexp.MustCompile("(.+) - Episode (\\d+) discussion")
	for _, post := range harvest.Posts {
		match := re.FindSubmatch([]byte(post.Title))
		if len(match) > 0 {
			writeComments(bot, post.Permalink)
			fmt.Println(string(match[1]), string(match[2]))
			fmt.Printf("[%s] posted [%s] [%s]\n", post.Author, post.Title, post.URL)
			break
		}
	}
}
