package main

import (
    "crypto/sha1"
    "fmt"
    "github.com/alphagov/router/trie"
    "github.com/Shopify/sarama"
    "hash"
    "net/http"
    "strings"
    "sync"
)

type Mux struct {
    mu         sync.RWMutex
    exactTrie  *trie.Trie
    prefixTrie *trie.Trie
    count      int
    checksum   hash.Hash
}

type muxEntry struct {
    prefix  bool
    handler http.Handler
}

func NewMux() *Mux {
    return &Mux{exactTrie: trie.NewTrie(), prefixTrie: trie.NewTrie(), checksum: sha1.New()}
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Pass event to central log
    mux.log(r.Method + " " + r.URL.Path)

    handler, ok := mux.lookup(r.URL.Path)

    if !ok {
        http.NotFound(w, r)
        return
    }

    handler.ServeHTTP(w, r)
}

func (mux *Mux) lookup(path string) (handler http.Handler, ok bool) {
    mux.mu.RLock()
    defer mux.mu.RUnlock()

    pathSegments := splitpath(path)
    val, ok := mux.exactTrie.Get(pathSegments)
    if !ok {
        val, ok = mux.prefixTrie.GetLongestPrefix(pathSegments)
    }
    if !ok {
        return nil, false
    }

    entry, ok := val.(muxEntry)
    if !ok {
        fmt.Printf("lookup: got value (%v) from trie that wasn't a muxEntry!", val)
        return nil, false
    }

    return entry.handler, ok
}

func (mux *Mux) addToStats(path string, prefix bool) {
    mux.count++
    mux.checksum.Write([]byte(path))
    if prefix {
        mux.checksum.Write([]byte("(true)"))
    } else {
        mux.checksum.Write([]byte("(false)"))
    }
}

func (mux *Mux) RouteCount() int {
    return mux.count
}

func (mux *Mux) RouteChecksum() []byte {
    return mux.checksum.Sum(nil)
}

func splitpath(path string) []string {
    partsWithBlanks := strings.Split(path, "/")

    parts := make([]string, 0, len(partsWithBlanks))
    for _, part := range partsWithBlanks {
            if part != "" {
                    parts = append(parts, part)
            }
    }

    return parts
}

func (mux *Mux) Handle(path string, prefix bool, handler http.Handler) {
    mux.mu.Lock()
    defer mux.mu.Unlock()

    mux.addToStats(path, prefix)
    if prefix {
        mux.prefixTrie.Set(splitpath(path), muxEntry{prefix, handler})
    } else {
        mux.exactTrie.Set(splitpath(path), muxEntry{prefix, handler})
    }
}

func (mux *Mux) log(path string) {
    var err error
    err = log.QueueMessage("route", sarama.StringEncoder("request.start"), sarama.StringEncoder(path))
    if err != nil {
        panic(err)
    }
}
