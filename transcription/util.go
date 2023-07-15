package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func parseData(data string) (string, string, string) {
	// Regular expression to extract start time, end time and text from the given data
	re := regexp.MustCompile(`^\[(\d{2}:\d{2}:\d{2}\.\d{3}) --> (\d{2}:\d{2}:\d{2}\.\d{3})\]\s+(.*)$`)
	matches := re.FindStringSubmatch(data)
	if matches == nil {
		return "", "", ""
	}
	start := matches[1]
	end := matches[2]
	text := matches[3]
	// Remove any unnecessary double quotes and white spaces from the text
	text = strings.Trim(text, "\" ")
	text = strings.ReplaceAll(text, "\"\"", "\"")
	return start, end, text
}

func SaveToCSV(data []string, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Comma = ','
	// Write header row
	header := []string{"start", "end", "text"}
	err = writer.Write(header)
	if err != nil {
		return err
	}
	// Write data rows
	for _, d := range data {
		start, end, text := parseData(d)
		if start == "" && end == "" && text == "" {
			continue
		}
		row := []string{start, end, text}
		err = writer.Write(row)
		if err != nil {
			return err
		}
	}
	writer.Flush()
	return nil
}

func clearCache(uuid, inputFileName string) error {

	err := os.Remove(fmt.Sprintf("cache/%s.csv", uuid))
	if err != nil {
		return fmt.Errorf("couldn't clear cache: %v", err)
	}
	err = os.Remove(fmt.Sprintf("cache/%s.wav", uuid))
	if err != nil {
		return fmt.Errorf("couldn't clear cache: %v", err)
	}
	return nil
}
