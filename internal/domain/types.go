package domain

type Message struct {
	Role    string
	Content string
}

type Session struct {
	ID string
}

type Config struct {
	ServerURL string
}
