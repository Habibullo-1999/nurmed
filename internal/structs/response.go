package structs

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Payload interface{} `json:"payload"`
}

func (r *Response) GetStatusCode() int {
	return r.Code
}
