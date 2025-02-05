package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/slack-go/slack"
)

var (
	slackAPIToken = os.Getenv("SLACK_API_TOKEN")

	channelName = flag.String("c", "", "channel name")
	reaction    = flag.String("r", "", "reaction")
	from        = flag.String("f", "", "from date (ISO8601)")
)

type Message struct {
	Timestamp   string             `json:"ts"`
	Text        string             `json:"text"`
	Attachments []slack.Attachment `json:"attachments"`
}

func main() {
	flag.Parse()
	if channelName == nil || *channelName == "" {
		log.Fatal("-c required")
	}
	if reaction == nil || *reaction == "" {
		log.Fatal("-r required")
	}
	if from == nil || *from == "" {
		*from = time.Now().AddDate(0, 0, -3).Format(time.RFC3339)
	}

	api := slack.New(slackAPIToken)

	channelID, err := getChannelID(api, *channelName)
	if err != nil {
		log.Fatalf("failed to get channel ID: %v", err)
	}

	messages, err := getMessagesWithReaction(api, channelID, *reaction, *from)
	if err != nil {
		log.Fatalf("failed to get messages: %v", err)
	}

	j, err := json.Marshal(messages)
	if err != nil {
		log.Fatalf("failed to marshal messages: %v", err)
	}
	fmt.Println(string(j))
}

func getChannelID(api *slack.Client, channelName string) (string, error) {
	conversationsParams := &slack.GetConversationsParameters{
		Types: []string{"public_channel"},
	}
	for {
		channels, cursor, err := api.GetConversations(conversationsParams)
		if err != nil {
			return "", err
		}

		for _, channel := range channels {
			if channel.Name == channelName {
				return channel.ID, nil
			}
		}

		if cursor != "" {
			conversationsParams.Cursor = cursor
		} else {
			break
		}
	}
	return "", fmt.Errorf("channel not found: %s", channelName)
}

func getMessagesWithReaction(api *slack.Client, channelID, reaction, from string) ([]Message, error) {
	fromTimestamp := convertToTimestamp(from)
	historyParams := slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Inclusive: false,
		Oldest:    fmt.Sprintf("%f", fromTimestamp),
	}
	var messages []Message

	for {
		history, err := api.GetConversationHistory(&historyParams)
		if err != nil {
			return nil, err
		}

		for _, message := range history.Messages {
			for _, reactionItem := range message.Reactions {
				if reactionItem.Name == reaction {
					messages = append(messages, Message{
						Timestamp:   message.Timestamp,
						Text:        message.Text,
						Attachments: message.Attachments,
					})
				}
			}
		}

		if history.HasMore {
			historyParams.Cursor = history.ResponseMetadata.Cursor
			continue
		}
		break
	}

	return messages, nil
}

func convertToTimestamp(datetime string) float64 {
	t, _ := time.Parse(time.RFC3339, datetime)
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9
}
