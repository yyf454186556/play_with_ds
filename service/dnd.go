package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

type DNDService struct {
	DNDMap   map[string][]*model.ChatCompletionMessage
	InputCh  chan string
	OutputCh chan string
}

func NewDNDService() *DNDService {
	return &DNDService{
		DNDMap:   make(map[string][]*model.ChatCompletionMessage),
		InputCh:  make(chan string),
		OutputCh: make(chan string),
	}
}

const (
	dnd_system_description    = "你是一个专业的DND的DM, 你的回答应该尽量保持DM的风格，不需要太多关于背景的描述，只需要作为一个DM进行流程的引导。你的回答应该尽量简洁。"
	polish_system_description = "你很擅长对汉语进行概括总结，现在你的任务是把这些对话进行概括总结，尽量保持简洁。尽量不要超过200字"
)

func (s *DNDService) DND() {
	client := GetClient()
	ctx := context.Background()
	message := make([]*model.ChatCompletionMessage, 0)
	message = append(message, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleSystem,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String(dnd_system_description),
		},
	})
	//reader := bufio.NewReader(os.Stdin)

	for {
		input := <-s.InputCh

		// 构造用户的输入
		message = append(message, &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(input),
			},
		})

		req := model.ChatCompletionRequest{
			Model:    "ep-20250211201454-gnlmx",
			Messages: message,
		}

		// 调用接口
		stream, err := client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			fmt.Printf("stream chat error: %v\n", err)
			return
		}

		// 流式获取响应
		reply := make([]string, 0)
		for {
			recv, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("Stream chat error: %v\n", err)
				break
			}

			if len(recv.Choices) > 0 {
				//fmt.Print(strings.TrimSpace(recv.Choices[0].Delta.Content))
				reply = append(reply, recv.Choices[0].Delta.Content)
			}
		}
		fmt.Println()
		s.OutputCh <- strings.Join(reply, "")

		// 基于LLM无状态，这里需要保留上下文
		message = append(message, &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleAssistant,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(strings.Join(reply, " ")),
			},
		})
		err = stream.Close()
		if err != nil {
			fmt.Printf("stream close error: %v\n", err)
		}

		if len(message) > 10 {
			message = s.DNDPolish(message)
		}
	}
}

func (s *DNDService) DNDPolish(msg []*model.ChatCompletionMessage) []*model.ChatCompletionMessage {
	client := GetClient()

	sb := strings.Builder{}
	for i := 1; i < len(msg); i++ {
		if msg[i].Role == model.ChatMessageRoleUser {
			sb.WriteString(fmt.Sprintf("冒险者说: %s \n", *msg[i].Content.StringValue))
		} else {
			sb.WriteString(fmt.Sprintf("DM说: %s \n", *msg[i].Content.StringValue))
		}
	}

	ctx := context.Background()
	message := make([]*model.ChatCompletionMessage, 0)
	message = append(message, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleSystem,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String(polish_system_description),
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
			StringValue: volcengine.String(fmt.Sprintf("%s 背景提要: %s", dnd_system_description, *resp.Choices[0].Message.Content.StringValue)),
		},
	})
	return newMessages
}

func (s *DNDService) DNDSocketfunc(c *gin.Context) {
	// 升级 HTTP 连接为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket 升级失败:", err)
		return
	}
	defer conn.Close()

	log.Println("客户端已连接")

	// 处理 WebSocket 消息
	for {
		// 读取客户端发送的消息
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取消息失败:", err)
			break
		}

		log.Printf("收到消息: %s", message)
		s.InputCh <- string(message)
		output := <-s.OutputCh

		// 将消息原样返回给客户端
		if err := conn.WriteMessage(messageType, []byte(strings.TrimSpace(output))); err != nil {
			log.Println("发送消息失败:", err)
			break
		}
	}

	log.Println("客户端已断开连接")
}

func (s *DNDService) DNDHandler(c *gin.Context) {
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
		delete(s.DNDMap, req.Name)
		c.JSON(http.StatusOK, gin.H{"error": "重置成功"})
		return
	}
	if _, ok := s.DNDMap[req.Name]; !ok {
		s.DNDMap[req.Name] = make([]*model.ChatCompletionMessage, 0)
	}
	if len(s.DNDMap[req.Name]) == 0 {
		s.DNDMap[req.Name] = append(s.DNDMap[req.Name], &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleSystem,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(dnd_system_description),
			},
		})
	}
	s.DNDMap[req.Name] = append(s.DNDMap[req.Name], &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleUser,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String(req.Content),
		},
	})

	err := s.AskDND(req.Name, s.DNDMap[req.Name])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	value := *s.DNDMap[req.Name][len(s.DNDMap[req.Name])-1].Content.StringValue
	if len(s.DNDMap[req.Name]) > 10 {
		s.DNDMap[req.Name] = s.DNDPolish(s.DNDMap[req.Name])
	}

	uuid := GetPicture(value)

	c.JSON(http.StatusOK, gin.H{"success": value, "uuid": uuid})
}

func (s *DNDService) AskDND(name string, msg []*model.ChatCompletionMessage) error {
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

	s.DNDMap[name] = msg

	return nil
}
