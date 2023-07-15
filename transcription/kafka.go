package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Shopify/sarama"
)

type Kafka struct {
	producer sarama.AsyncProducer
	consumer sarama.Consumer
}

func NewKafkaProducer() (*Kafka, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Wait for all in-sync replicas to acknowledge
	config.Producer.Compression = sarama.CompressionSnappy   // Use snappy compression
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms
	config.Producer.Return.Successes = true

	producer, err := sarama.NewAsyncProducer([]string{"broker:29092"}, config)
	if err != nil {
		return nil, err
	}

	kafka := &Kafka{
		producer: producer,
	}

	go kafka.handleSuccesses()

	return kafka, nil
}

func (k *Kafka) handleSuccesses() {
	for {
		select {
		case <-k.producer.Successes():
			log.Print("SUCC: Published message")
		case err := <-k.producer.Errors():
			log.Print("ERR: Failed to produce message", err)
		}
	}
}

func (k *Kafka) ProduceMessage(topic string, value interface{}) {
	messageBytes, err := json.Marshal(value)
	if err != nil {
		log.Print("ERR: ", err)
		return
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(messageBytes),
	}

	k.producer.Input() <- message
}
