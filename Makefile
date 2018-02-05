issuebot: FORCE clean
	vndr
	go build issuebot.go

clean:
	rm -f issuebot

FORCE:
