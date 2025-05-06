package messages

type Message struct {
	Subject string
	Body    string
	Errors  []error
}

type Sender interface {
	Send(msg Message) error
}
