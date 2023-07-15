FROM golang:alpine

RUN apk update && apk upgrade
RUN apk add ffmpeg 

RUN mkdir /app

WORKDIR /app

ADD go.mod .
ADD go.sum .

RUN go mod download
ADD . .

RUN go install -mod=mod github.com/githubnemo/CompileDaemon

ENTRYPOINT CompileDaemon --build="go build" --command=./audio-extraction