package service

type Ask20Request struct {
	Auth    string `json:"auth"`
	Name    string `json:"name"`
	Content string `json:"content"`
	StoryID int    `json:"story_id"`
}
type CommonRequest struct {
	Auth    string `json:"auth"`
	Content string `json:"content"`
	StoryID int    `json:"story_id"`
}

type AddRoleRequest struct {
	Auth        string `json:"auth"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
