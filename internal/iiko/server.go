package iiko

type Server struct {
	Host string
	Port string
}

func NewServer(host, port string) *Server {
	return &Server{
		Host: host,
		Port: port,
	}
}
