package main

import (
	"fmt"
	"regexp"

	"github.com/turnage/graw/reddit"
)

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

	counter := 0
	for _, post := range harvest.Posts {
		if match, _ := regexp.MatchString(".+ - Episode \\d+ discussion", post.Title); match {
			fmt.Println(match)
			fmt.Printf("[%s] posted [%s] [%s]\n", post.Author, post.Title, post.URL)
		}
		counter++
		if counter == 1000 {
			break
		}
	}
}
