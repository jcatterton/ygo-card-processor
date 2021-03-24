package producer

import (
	"encoding/json"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sirupsen/logrus"

	"ygo-card-processor/models"
)

type KafkaProducer interface {
	Produce(event string, message string, isError bool)
}

type Producer struct {
	Producer        *kafka.Producer
	Topic           string
	DeliveryChannel chan kafka.Event
}

func CreateProducer(broker string, topic string) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
	})
	if err != nil {
		return nil, err
	}

	producer := Producer{
		Producer:        p,
		Topic:           topic,
		DeliveryChannel: make(chan kafka.Event),
	}

	return &producer, nil
}

func (p *Producer) Produce(event string, message string, isError bool) {
	log := models.Log{
		AppName:   "ygo-card-processor",
		Event:     event,
		Message:   message,
		IsError:   isError,
		TimeStamp: time.Now(),
	}

	bytes, err := json.Marshal(log)
	if err != nil {
		logrus.WithError(err).Error("Error creating log")
	}

	if err := p.produce(bytes); err != nil {
		logrus.WithError(err).Error("Error producing kafka message")
	}
}

func (p *Producer) produce(message []byte) error {
	kMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          message,
	}

	return p.Producer.Produce(kMessage, p.DeliveryChannel)
}
