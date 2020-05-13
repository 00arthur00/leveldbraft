package config

type Config struct {
	DataDir     string
	HTTPAddr    string
	RaftTCPAddr string
	Bootstrap   bool
	JoinAddr    string
}
