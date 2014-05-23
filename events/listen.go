package events

import (
	"bytes"
	"fmt"
	"os"

	"code.google.com/p/goprotobuf/proto"

	"api.umn.edu/route/cache"
	"api.umn.edu/route/interfaces"
	"api.umn.edu/route/logger"
	"api.umn.edu/route/util"
	zmq "github.com/alecthomas/gozmq"
)

var topics = []string{"auth", "route"}

func Listen() {
	/* Create logger */
	logging := logger.NewLogger("route.event.listen", logger.Console)

	/* Connect to cache */
	local_cache, err := cache.NewCache(cache.Redis)
	if err != nil {
		logging.Log("internal", "route.error", "failed to bind to redis", "[fg-red]")
		os.Exit(1)
	}

	/* Set up event listener */
	context, err := zmq.NewContext()
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ context", "[fg-red]")
		return
	}

	defer context.Close()

	s, err := context.NewSocket(zmq.SUB)
	if err != nil {
		logging.Log("internal", "route.error", "failed to create ZMQ socket", "[fg-red]")
		return
	}

	s.Connect(util.GetenvDefault("EVENT_BIND", "tcp://127.0.0.1:6666"))

	defer s.Close()

	/* Subscribe to event topics */
	for _, topic := range topics {
		s.SetSubscribe(topic)
	}

	/* Listen loop */
	for {
		message, _ := s.RecvMultipart(0)

		/* Extract message parts */
		topic := message[0]
		body := message[1]

		switch {
		case bytes.Equal(topic, []byte(topics[0])):
			/* Auth message */
			data := &interfaces.Auth{}

			/* Extract message into structure */
			err := proto.Unmarshal(body, data)

			if err != nil {
				logging.Log("internal", "route.error", "failed demarshall message", "[fg-red]")
				return
			}

			/* Store in appropriate collection based on topic */
			public_key := data.GetPublicKey()
			private_key := data.GetPrivateKey()
			fmt.Println(public_key)
			fmt.Println(private_key)

			payload := map[string]string{
				public_key: private_key,
			}

			local_cache.Set(fmt.Sprintf("key:%s", public_key), payload)

			logging.Log("internal", "route.event", "key added to cache", "[fg-green]")
		}
	}
}
