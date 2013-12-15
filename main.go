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
    var config Config
    gcfg.ReadFileInto(&config, "hosts.conf")

    mux := NewMux()

    // Create route handlers
    for host, label := range config.Host {
        url, _ := url.Parse(host)
        proxy := httputil.NewSingleHostReverseProxy(url)
        mux.Handle("/" + label.Label, true, proxy)
    }

    err = log.SendMessage("internal", nil, sarama.StringEncoder("Router started"))

    http.ListenAndServe(":9343", mux)
}
