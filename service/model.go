package service

type CommonRequest struct {
	Auth    string `json:"auth"`
	Name    string `json:"name"`
	Content string `json:"content"`
}
