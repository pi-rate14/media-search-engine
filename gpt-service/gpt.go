package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type GPT struct {
	gptKey string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Choices []Choice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func NewGPT(key string) *GPT {
	return &GPT{
		gptKey: key,
	}
}

func (gpt *GPT) buildPrompt(query, context string) string {
	return fmt.Sprintf("Answer the question in detail with the help of the given context.\nQuestion: %s\nContext:%s", query, context)
}

func (gpt *GPT) makeRequest(prompt string) ChatCompletionResponse {
	url := "https://api.openai.com/v1/chat/completions"

	messages := []Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	requestBody, _ := json.Marshal(map[string]interface{}{
		"model":    "gpt-3.5-turbo",
		"messages": messages,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gpt.gptKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var response ChatCompletionResponse
	json.NewDecoder(resp.Body).Decode(&response)

	return response
}
