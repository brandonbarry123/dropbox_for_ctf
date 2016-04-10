// SUPPORT CODE
//
// You shouldn't need to alter
// the contents of this file

package rpcType

type Server struct {
	Callback func(req Request, resp *Response) error
}

type Request struct {
	Name string
	Args [][]byte
}

type Response struct {
	Return []byte
}

func (s *Server) Request(req Request, resp *Response) error {
	return s.Callback(req, resp)
}
