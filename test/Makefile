all: ghfetch ghmod dumbbot

ghfetch: ../github/github.go ghfetch.go
	go build ghfetch.go

ghmod: ../github/github.go ghmod.go
	go build ghmod.go

dumbbot: ../github/github.go ../slack/slackbot.go dumbbot.go
	go build dumbbot.go

clean:
	rm -f ghfetch ghmod dumbbot
