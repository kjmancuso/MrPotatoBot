package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/nlopes/slack"
)

// GithubWebhookSecret For verifying webhook payload
var GithubWebhookSecret string

// SlackClient to do slack things
var SlackClient *slack.Client

// SlackUsers a mapping of slackusers, since they dont map to usernames anymore
// This is naive, and should be optimized
var SlackUsers map[string]string

func slackPopulateMap() {
	SlackUsers = make(map[string]string)
	users, err := SlackClient.GetUsers()
	for _, element := range users {
		SlackUsers[element.Name] = element.ID
	}

	if err != nil {
		fmt.Printf("Error populating slacklist: %s", err)
	}
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(GithubWebhookSecret))
	if err != nil {
		log.Printf("error validating request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("could not parse webhook: err=%s\n", err)
		return
	}

	switch e := event.(type) {
	case *github.PullRequestEvent:
		for _, element := range e.PullRequest.RequestedReviewers {
			slackNotifyReviewer(strings.Replace(github.Stringify(element.Login), "\"", "", -1),
				strings.Replace(github.Stringify(e.Sender.Login), "\"", "", -1))
		}
	default:
		log.Printf("unknown event type %s\n", github.WebHookType(r))
		return
	}
}

func slackNotifyReviewer(uid string, sender string) {
	var channelID = SlackUsers[uid]
	uid, _, err := SlackClient.PostMessage(channelID,
		slack.MsgOptionText(fmt.Sprintf("Your attention has been requested to a pull request by %s!!", sender), false),
		slack.MsgOptionAsUser(true),
		[]slack.AttachmentField
			{
				Title: "one",
			Value: "value",
			Short: true,
			}
	)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	fmt.Printf("%s", uid)
}
func main() {
	secret, ok := os.LookupEnv("GITHUB_WEBHOOK_SECRET")
	if !ok {
		fmt.Println("GITHUB_WEBHOOK_SECRET not set, webhook payload verification disabled")
		GithubWebhookSecret = ""
	} else {
		GithubWebhookSecret = secret
	}
	slackToken, ok := os.LookupEnv("SLACK_TOKEN")
	if !ok {
		fmt.Println("!FATAL! SLACK_TOKEN not set.")
		os.Exit(1)
	}
	SlackClient = slack.New(slackToken)
	slackPopulateMap()
	log.Println("server started")
	http.HandleFunc("/webhook", handleWebhook)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
