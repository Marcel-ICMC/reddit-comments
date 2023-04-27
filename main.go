package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Marcel-ICMC/graw/reddit"
)

var Logger *log.Logger
var errorLogger *log.Logger

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
	chunksize := 100

	for _, more := range chunksBy(moreChildren, chunksize) {
		go func(more []string) {
			harvest, err := bot.ListingWithParams(
				"/api/morechildren",
				map[string]string{
					"link_id":  "t3_" + postID,
					"children": strings.Join(more, ","),
				},
			)
			if err != nil {
				errorLogger.Println("Failed while fetching more comments: ", err)
			} else {
				chn <- harvest
			}
		}(more)
	}

	for i := 0; i < int(math.Ceil(float64(len(moreChildren))/float64(chunksize))); i++ {
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
	// solving comment tree
	for len(queue) != 0 {
		comment := queue[0]
		queue = queue[1:]

		comment.Replies = reply_tree[comment]
		queue = append(queue, reply_tree[comment]...)
	}

	return
}

func getAllComments(bot reddit.Bot, post *reddit.Post) {
	var queue []*reddit.Comment
	requests := 0
	comment_count := 0

	if post.More != nil && post.More.Children != nil {
		requests += (len(post.More.Children) / 100) + 1
		post.Replies = append(post.Replies, getMoreComments(bot, post.ID, post.More.Children)...)
	}
	queue = append(queue, post.Replies...)
	comment_count += len(post.Replies)

	for len(queue) != 0 {
		r := queue[0]
		queue = queue[1:]
		if r.More != nil {
			requests += (len(r.More.Children) / 100) + 1
			r.Replies = append(r.Replies, getMoreComments(bot, post.ID, r.More.Children)...)
		}
		queue = append(queue, r.Replies...)
		comment_count += len(r.Replies)
	}

	Logger.Printf("%d comments, needed %d more comments requests", comment_count, requests)
}

func threadToJson(bot reddit.Bot, permalink string) ([]byte, error) {
	Logger.Println(permalink)
	post, _ := bot.Thread(permalink)
	getAllComments(bot, post)
	thread_json, err := json.Marshal(post)

	return thread_json, err
}

func jsonToFile(thread_json []byte, file_path string) error {
	f, err := os.Create(file_path)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	nbytes, err := f.Write(thread_json)
	if err != nil {
		errorLogger.Println("Failed to write to file ", file_path, ": ", err)
	}
	Logger.Printf("wrote %d bytes to %s", nbytes, file_path)
	return nil
}

func main() {
	f, err := os.OpenFile(
		fmt.Sprintf("logs/%s.txt", time.Now().Format(time.DateTime)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	Logger = log.New(f, "INFO: ", log.Ldate|log.Lmicroseconds|log.Lshortfile)
	errorLogger = log.New(f, "ERROR: ", log.Ldate|log.Lmicroseconds|log.Lshortfile)

	bot, err := reddit.NewBotFromAgentFile("lurker-bot.agent", 0)
	if err != nil {
		errorLogger.Println("Failed to create bot handle: ", err)
		return
	}
	var after string = "t3_12jtvwr"

	for {
		Logger.Printf("Current after is %s\n", after)
		harvest, err := bot.ListingWithParams(
			"/r/anime/new",
			map[string]string{
				"after": after,
			},
		)
		if err != nil {
			errorLogger.Println("Failed to fetch /r/anime: ", err)
			return
		}

		re := regexp.MustCompile("(.+) - Episode (\\d+) discussion")
		for _, post := range harvest.Posts {
			after = post.Name
			match := re.FindSubmatch([]byte(post.Title))
			if len(match) > 0 {
				write_path := fmt.Sprintf("%s %s.json", match[1], match[2])
				write_path = strings.ReplaceAll(write_path, "/", "-")
				write_path = "threads/" + write_path
				if _, err := os.Stat(write_path); err == nil {
					Logger.Printf("File %s already exists", write_path)
					continue
				}

				Logger.Printf("[%s] posted [%s] [%s]\n", post.Author, post.Title, post.URL)

				thread_json, _ := threadToJson(bot, post.Permalink)
				jsonToFile(thread_json, write_path)
				break
			}
		}
	}
}
