package queue

type ProducerConfig struct {
	Brokers []string
	Topic   string
}

type ConsumerConfig struct {
	Brokers        []string
	Topic          string
	GroupID        string
	ConsumerOffset int64
}
