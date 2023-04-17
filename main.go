package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/Marcel-ICMC/graw/reddit"
)

var Logger *log.Logger

func chunksBy[T any](items []T, chunk_size int) (chunks [][]T) {
	for chunk_size < len(items) {
		items, chunks = items[chunk_size:], append(chunks, items[0:chunk_size:chunk_size])
	}
	return append(chunks, items)
}

func getMoreComments(bot reddit.Bot, postID string, moreChildren []string) (replies []*reddit.Comment) {
	reply_tree := make(map[*reddit.Comment][]*reddit.Comment)
	id_to_comment := make(map[string]*reddit.Comment)
	queue := make([]*reddit.Comment, 0)

	chn := make(chan reddit.Harvest, 70)

	Logger.Printf("Len(moreChildren) = %d\n", len(moreChildren))

	for _, more := range chunksBy(moreChildren, 100) {
		go func(more []string) {
			harvest, err := bot.ListingWithParams(
				"/api/morechildren.json",
				map[string]string{
					"link_id":  "t3_" + postID,
					"children": strings.Join(more, ","),
				},
			)
			if err != nil {
				panic(err)
			}
			chn <- harvest
		}(more)
	}

	for i := 0; i < int(math.Ceil(float64(len(moreChildren))/100)); i++ {
		harvest := <-chn
		for _, harvested := range harvest.Comments {
			id_to_comment[harvested.Name] = harvested
		}
	}

	for _, harvested := range id_to_comment {
		if value, ok := id_to_comment[harvested.ParentID]; ok {
			reply_tree[value] = append(reply_tree[value], harvested)
		} else {
			queue = append(queue, harvested)
		}
	}

	replies = append(replies, queue...)
	Logger.Println("Solving more comments tree")
	// solving comment tree
	for len(queue) != 0 {
		comment := queue[0]
		queue = queue[1:]

		comment.Replies = reply_tree[comment]
		queue = append(queue, reply_tree[comment]...)
	}

	Logger.Println("Solved comment tree")
	return
}

func getAllComments(bot reddit.Bot, post *reddit.Post) {
	var queue []*reddit.Comment
	queue = append(queue, post.Replies...)

	for len(queue) != 0 {
		Logger.Printf("Len(queue) = %d\n", len(queue))
		r := queue[0]
		queue = queue[1:]
		if r.More != nil {
			Logger.Println("Getting more comments")
			r.Replies = append(r.Replies, getMoreComments(bot, post.ID, r.More.Children)...)
		}
		queue = append(queue, r.Replies...)
	}
}

func threadToJson(bot reddit.Bot, permalink string) ([]byte, error) {
	Logger.Println(permalink)
	post, _ := bot.Thread(permalink)
	getAllComments(bot, post)
	thread_json, error := json.Marshal(post)

	return thread_json, error
}

func jsonToFile(thread_json []byte, file_path string) error {
	f, err := os.Create(file_path)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	nbytes, err := f.Write(thread_json)
	if err != nil {
		return err
	}
	Logger.Printf("wrote %d bytes\n", nbytes)
	return nil
}

func main() {
	Logger = log.New(os.Stderr, "", log.Ldate|log.Lmicroseconds|log.Lshortfile)
	bot, err := reddit.NewBotFromAgentFile("lurker-bot.agent", 0)
	if err != nil {
		Logger.Println("Failed to create bot handle: ", err)
		return
	}
	var after string = ""

	for {
		Logger.Printf("Current after is %s\n", after)
		harvest, err := bot.Listing("/r/anime", after)
		if err != nil {
			Logger.Println("Failed to fetch /r/anime: ", err)
			return
		}

		re := regexp.MustCompile("(.+) - Episode (\\d+) discussion")
		for _, post := range harvest.Posts {
			match := re.FindSubmatch([]byte(post.Title))
			if len(match) > 0 {
				after = "t1_" + post.ID
				Logger.Printf("[%s] posted [%s] [%s]\n", post.Author, post.Title, post.URL)
				Logger.Println(string(match[1]), string(match[2]))

				thread_json, _ := threadToJson(bot, post.Permalink)
				jsonToFile(thread_json, fmt.Sprintf("threads/%s%s.json", match[1], match[2]))
			}
		}
	}
}
