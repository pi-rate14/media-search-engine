FROM golang:alpine

RUN apk add --no-cache git

RUN mkdir /app
WORKDIR /app

# Clone the repository
RUN git clone https://github.com/ggerganov/whisper.cpp.git whisper-repo

WORKDIR /app/whisper-repo

# Run download-ggml-model.sh script
RUN apk update \                                                                                                                                                                                                                        
  && apk add ca-certificates wget \                                                                                                                                                                                                      
  && update-ca-certificates
RUN apk add --no-cache bash
RUN bash ./models/download-ggml-model.sh base.en
RUN apk add --no-cache make
RUN apk add --no-cache g++
RUN make

WORKDIR /app

# Build the server
COPY go.mod ./
COPY go.sum ./  
RUN go mod download
COPY . ./

RUN go install -mod=mod github.com/githubnemo/CompileDaemon

EXPOSE 8085

ENTRYPOINT CompileDaemon --build="go build" --command=./transcription
