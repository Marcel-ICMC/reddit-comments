package main

import (
	"fmt"
	"regexp"

	"github.com/turnage/graw/reddit"
)

func countComments(replies []*reddit.Comment) int {
	counter := len(replies)
	for _, r := range replies {
		fmt.Println("Replies: ", len(r.Replies), " Body:", r.Body)
		counter += countComments(r.Replies)
	}

	return counter
}

func getComments(bot reddit.Bot, permalink string) {
	post, _ := bot.Thread(permalink)
	fmt.Println("comment count: ", countComments(post.Replies))
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
			getComments(bot, post.Permalink)
			fmt.Println(string(match[1]), string(match[2]))
			fmt.Printf("[%s] posted [%s] [%s]\n", post.Author, post.Title, post.URL)
			break
		}
	}
}
