all: ghfetch ghmod

ghfetch: ../github/github.go ghfetch.go
	go build ghfetch.go

ghmod: ../github/github.go ghmod.go
	go build ghmod.go

clean:
	rm -f ghfetch ghmod
