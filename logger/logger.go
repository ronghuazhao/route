package logger

import (
    "fmt"
    "time"
    "sync"
	"github.com/Shopify/sarama"
)

const (
    Console = iota
    Kafka
)

type Logger struct {
	mu    sync.RWMutex
	producer    *sarama.Producer
	handler     int
}

func NewLogger(name string, handler int) *Logger {

    logger := &Logger{}

    if handler == Kafka {
        // Create Kafka connection
        client, err := sarama.NewClient(name, []string{"localhost:9092"}, &sarama.ClientConfig{
            MetadataRetries:  10,
            WaitForElection:  250 * time.Millisecond})
        if err != nil {
            fmt.Println(err)
            return logger.fallback(logger)
        } else {
            // defer client.Close()
        }

        producer, err := sarama.NewProducer(client, &sarama.ProducerConfig{
            RequiredAcks:     sarama.WaitForLocal,
            MaxBufferedBytes: 4096,
            MaxBufferTime:    4096})

        if err != nil {
            return logger.fallback(logger)
        } else {
            // defer producer.Close()

            logger.producer = producer
            logger.handler = Kafka
        }
    } else if handler == Console {
        fmt.Println("> internal: using console output")
        logger.handler = Console
    }

	return logger
}

func (logger *Logger) fallback(instance *Logger) *Logger {
    fmt.Println("> internal: central log unavailable")
    fmt.Println("> internal: using console output")
    instance.handler = Console
    return instance
}

func (logger *Logger) Log(topic string, key string, value string) {
	logger.mu.RLock()
	defer logger.mu.RUnlock()

    if logger.handler == Kafka {
        logger.producer.QueueMessage(topic, sarama.StringEncoder(key), sarama.StringEncoder(value))
    } else if logger.handler == Console {
        fmt.Printf("> %s: %s\n", topic, value)
    }
}
