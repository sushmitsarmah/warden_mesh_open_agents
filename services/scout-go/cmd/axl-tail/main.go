package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/local/swarm/scout/pkg/axl"
)

func main() {
	var topicFilter string
	flag.StringVar(&topicFilter, "topic", "", "filter to a single topic")
	flag.Parse()

	node, err := axl.NewNode(axl.LoadBootstrapFromEnv())
	if err != nil {
		slog.Error("axl node", "err", err)
		os.Exit(1)
	}
	defer node.Close()

	topics := []string{
		"targets/discovered",
		"analysis/findings",
		"exploit/verified",
		"disclosure/published",
	}

	for _, t := range topics {
		if topicFilter != "" && t != topicFilter {
			continue
		}
		ch, err := node.Subscribe(t)
		if err != nil {
			slog.Error("subscribe", "topic", t, "err", err)
			continue
		}
		go func(topic string, c <-chan []byte) {
			for msg := range c {
				var pretty bytes.Buffer
				_ = json.Indent(&pretty, msg, "", "  ")
				fmt.Printf("[%s] %s %s\n", topic, time.Now().Format(time.RFC3339), pretty.String())
			}
		}(t, ch)
	}

	select {}
}
