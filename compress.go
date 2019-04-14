package server

type (
	Compress struct {
		Types []string `json:"types,omitempty"`
	}
)

func (config *Compress) init(server *Server, handler *Handler) {
	if config.Types == nil {
		config.Types = []string{"application/json", "text/plain"}
	}
}
