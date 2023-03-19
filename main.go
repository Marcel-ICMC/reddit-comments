package main

import (
        "fmt"

        "github.com/turnage/graw/reddit"
)

func main() {
        bot, err := reddit.NewBotFromAgentFile("lurker-bot.agent", 0)
        if err != nil {
                fmt.Println("Failed to create bot handle: ", err)
                return
        }

        harvest, err := bot.Listing("/r/golang", "")
        if err != nil {
                fmt.Println("Failed to fetch /r/golang: ", err)
                return
        }

        for _, post := range harvest.Posts[:5] {
                fmt.Printf("[%s] posted [%s]\n", post.Author, post.Title)
        }
}