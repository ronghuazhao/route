package main

import (
    "code.google.com/p/gcfg"
    "github.com/Shopify/sarama"
    "net/http"
    "net/http/httputil"
    "net/url"
    "runtime"
)

// Create global log instance
var log *sarama.Producer

func Log(topic string, key string, value string) {
    log.QueueMessage(topic, sarama.StringEncoder(key), sarama.StringEncoder(value))
}

func main() {

    // Use all cores
    runtime.GOMAXPROCS(runtime.NumCPU())

    // Create Kafka connection
    kafka, err := sarama.NewClient("broadcast", []string{"localhost:9092"}, nil)
    if err != nil {
        panic(err)
    }
    defer kafka.Close()

    producer, err := sarama.NewProducer(kafka, &sarama.ProducerConfig{
                RequiredAcks: sarama.WaitForLocal,
                MaxBufferedBytes: 4096,
                MaxBufferTime: 4096})
    if err != nil {
        panic(err)
    }
    defer producer.Close()

    // Bind logger globally
    log = producer

    // Host config file structure
    type Config struct {
        Host map[string]*struct {
            Label string
        }
    }

    // Read in host file
    var hosts Config
    gcfg.ReadFileInto(&hosts, "hosts.conf")

    mux := NewMux()

    // Create route handlers
    for host, conf := range hosts.Host {
        url, _ := url.Parse(host)

        domain := url.Host
        label := conf.Label

        proxy := httputil.NewSingleHostReverseProxy(url)
        prefix := "/" + label

        mux.Register(label, domain, prefix, proxy)
    }

    Log("internal", "route.start", "Router started")

    http.ListenAndServe(":9343", mux)
}
