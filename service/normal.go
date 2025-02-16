package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

type NormalService struct {
	NormalMap map[string][]*model.ChatCompletionMessage
	InputCh   chan string
	OutputCh  chan string
}

func NewNormalService() *NormalService {
	return &NormalService{
		NormalMap: make(map[string][]*model.ChatCompletionMessage),
		InputCh:   make(chan string),
		OutputCh:  make(chan string),
	}
}

func (s *NormalService) NormalDSHandler(c *gin.Context) {
	req := &CommonRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Auth != "zzyztyy" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth error"})
		return
	}
	if req.Content == "重置" {
		delete(s.NormalMap, req.Name)
		c.JSON(http.StatusOK, gin.H{"error": "重置成功"})
		return
	}
	if _, ok := s.NormalMap[req.Name]; !ok {
		s.NormalMap[req.Name] = make([]*model.ChatCompletionMessage, 0)
	}
	if len(s.NormalMap[req.Name]) == 0 {
		s.NormalMap[req.Name] = append(s.NormalMap[req.Name], &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleSystem,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String("你是deepseek-v3, 强大的AI，请帮忙解决人类的问题"),
			},
		})
	}
	s.NormalMap[req.Name] = append(s.NormalMap[req.Name], &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleUser,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String(req.Content),
		},
	})

	err := s.AskNormal(req.Name, s.NormalMap[req.Name])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	value := *s.NormalMap[req.Name][len(s.NormalMap[req.Name])-1].Content.StringValue
	if len(s.NormalMap[req.Name]) > 10 {
		s.NormalMap[req.Name] = s.NormalPolish(s.NormalMap[req.Name])
	}

	c.JSON(http.StatusOK, gin.H{"success": value})
}

func (s *NormalService) AskNormal(name string, msg []*model.ChatCompletionMessage) error {
	client := GetClient()
	ctx := context.Background()

	req := model.ChatCompletionRequest{
		Model:    "ep-20250211201454-gnlmx",
		Messages: msg,
	}

	// 调用接口
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return err
	}

	// 基于LLM无状态，这里需要保留上下文
	msg = append(msg, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleAssistant,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String(*resp.Choices[0].Message.Content.StringValue),
		},
	})

	//if len(msg) > 10 {
	//	msg = NormalPolish(msg)
	//}
	s.NormalMap[name] = msg

	return nil
}

func (s *NormalService) NormalPolish(msg []*model.ChatCompletionMessage) []*model.ChatCompletionMessage {
	client := GetClient()

	sb := strings.Builder{}
	for i := 1; i < len(msg); i++ {
		if msg[i].Role == model.ChatMessageRoleUser {
			sb.WriteString(fmt.Sprintf("用户说: %s \n", *msg[i].Content.StringValue))
		} else {
			sb.WriteString(fmt.Sprintf("Deepseek说: %s \n", *msg[i].Content.StringValue))
		}
	}

	ctx := context.Background()
	message := make([]*model.ChatCompletionMessage, 0)
	message = append(message, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleSystem,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String("你很擅长对汉语进行概括总结，现在你的任务是把这些对话进行概括总结，尽量保持简洁。尽量不要超过200字"),
		},
	})
	message = append(message, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleUser,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String(sb.String()),
		},
	})

	req := model.ChatCompletionRequest{
		Model:    "ep-20250211201454-gnlmx",
		Messages: message,
	}
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		fmt.Println("CreateChatCompletion error: ", err)
		return msg
	}
	fmt.Println("Polish: " + *resp.Choices[0].Message.Content.StringValue)

	newMessages := make([]*model.ChatCompletionMessage, 0)
	newMessages = append(newMessages, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleSystem,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String(fmt.Sprintf("你是deepseek-v3, 强大的AI，请帮忙解决人类的问题。背景提要: %s", *resp.Choices[0].Message.Content.StringValue)),
		},
	})
	return newMessages
}
