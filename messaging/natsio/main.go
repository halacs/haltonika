package main

/*
persistent publish-subscribe with nats.io to ensure multithreading and fault tolerance
*/
/*
func main() {
	timeout := 2 * time.Second

	// Connect to NATS
	nc, _ := nats.Connect(nats.DefaultURL)

	// Create JetStream Context
	js, _ := nc.JetStream(nats.PublishAsyncMaxPending(256))

	// Simple Stream Publisher
	js.Publish("ORDERS.scratch", []byte("hello"))

	// Simple Async Stream Publisher
	for i := 0; i < 500; i++ {
		js.PublishAsync("ORDERS.scratch", []byte("hello"))
	}
	select {
	case <-js.PublishAsyncComplete():
	case <-time.After(5 * time.Second):
		fmt.Println("Did not resolve in time")
	}

	// Simple Async Ephemeral Consumer
	js.Subscribe("ORDERS.*", func(m *nats.Msg) {
		fmt.Printf("Received a JetStream message: %s\n", string(m.Data))
	})

	// Simple Sync Durable Consumer (optional SubOpts at the end)
	sub, err := js.SubscribeSync("ORDERS.*", nats.Durable("MONITOR"), nats.MaxDeliver(3))
	m, err := sub.NextMsg(timeout)

	// Simple Pull Consumer
	sub, err := js.PullSubscribe("ORDERS.*", "MONITOR")
	msgs, err := sub.Fetch(10)

	// Unsubscribe
	sub.Unsubscribe()

	// Drain
	sub.Drain()
}
*/
