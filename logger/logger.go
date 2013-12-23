package logger

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/foize/go.sgr"
	"sync"
	"time"
)

const (
	Console = iota
	Kafka
)

type Logger struct {
	mu       sync.RWMutex
	producer *sarama.Producer
	handler  int
}

func NewLogger(name string, handler int) *Logger {

	logger := &Logger{}

	if handler == Kafka {
		// Create Kafka connection
		client, err := sarama.NewClient(name, []string{"localhost:9092"}, &sarama.ClientConfig{
			MetadataRetries: 10,
			WaitForElection: 250 * time.Millisecond})
		if err != nil {
			fmt.Println(err)
			return logger.fallback(logger)
        }

		producer, err := sarama.NewProducer(client, &sarama.ProducerConfig{
			RequiredAcks:     sarama.WaitForLocal,
			MaxBufferedBytes: 4096,
			MaxBufferTime:    4096})

		if err != nil {
			return logger.fallback(logger)
		} else {
			logger.producer = producer
			logger.handler = Kafka
		}
	} else if handler == Console {
		logger.handler = Console
        logger.Log("internal", "router.status", "using console output", "[fg-blue]")
	}

	return logger
}

func (logger *Logger) fallback(instance *Logger) *Logger {
	instance.handler = Console
	instance.Log("internal", "router.status", "central log unavailable", "[fg-blue]")
	instance.Log("internal", "router.status", "using console output", "[fg-blue]")
	return instance
}

func (logger *Logger) Log(topic string, key string, value string, formatting interface{}) {
	logger.mu.RLock()
	defer logger.mu.RUnlock()

	if logger.handler == Kafka {
		logger.producer.QueueMessage(topic, sarama.StringEncoder(key), sarama.StringEncoder(value))
	} else if logger.handler == Console {
	    var message string

	    if formatting != nil {
            message = fmt.Sprintf("%s> %s: %s", formatting, topic, value)
        } else {
            message = fmt.Sprintf("> %s: %s", topic, value)
        }

        message = sgr.MustParseln(message)
		fmt.Print(message)
	}
}
