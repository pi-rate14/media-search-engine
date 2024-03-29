package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Query struct {
	QueryString string `json:"query"`
	Uuid        string `json:"uuid"`
}

type ContextResponse struct {
	Context string `json:"context"`
}

type QueryResponse struct {
	Data string `json:"data"`
}

func (app *app) QueryGPT(w http.ResponseWriter, r *http.Request) {
	var q Query

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Do something with the Person struct...
	// var client = &http.Client{Timeout: 10 * time.Second}
	queryJson, err := json.Marshal(q)
	if err != nil {
		http.Error(w, "cant marshal body", http.StatusInternalServerError)
	}

	resp, err := http.Post("http://embedding-generation:8090/context", "application/json",
		bytes.NewBuffer(queryJson),
	)

	if err != nil {
		http.Error(w, "cant get context", http.StatusInternalServerError)
	}

	var c ContextResponse
	fmt.Println(resp.Body)
	err = json.NewDecoder(resp.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println(c)
	fmt.Println(app.gpt.buildPrompt(q.QueryString, c.Context))

	gptResponse := app.gpt.makeRequest(app.gpt.buildPrompt(q.QueryString, c.Context))
	var response QueryResponse
	response.Data = gptResponse.Choices[0].Message.Content
	// fmt.Println(gptResponse.Choices[0].Message.Content)
	// Convert the response variable to a JSON-encoded byte array
	// var response QueryResponse
	// response.Data = c.Context
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the Content-Type header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response to the HTTP client
	w.Write(jsonResponse)

}
