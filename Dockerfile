FROM golang:latest

COPY ./issuebot /usr/local/bin

EXPOSE 80

ENTRYPOINT ["issuebot"]
