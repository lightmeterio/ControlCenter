package notification

type Content interface {
}

type Notification struct {
	ID      uint64
	Content Content
}

type Center interface {
	Notify(Content)
}
