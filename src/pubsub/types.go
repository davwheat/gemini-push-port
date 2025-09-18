package pubsub

type Destination struct {
	Name            string `json:"name"`
	DestinationType string `json:"destinationType"`
}

type PushPortSequence struct {
	SequenceId string `json:"string"`
}

type Properties struct {
	PushPortSequence PushPortSequence `json:"PushPortSequence"`
}

type WrappedMessage struct {
	Partition int    `json:"partition"`
	Message   string `json:"message"`
}
