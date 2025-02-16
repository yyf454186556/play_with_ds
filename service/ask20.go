package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有跨域请求
		return true
	},
}

type Ask20Service struct {
	Ask20Map map[string][]*model.ChatCompletionMessage
	InputCh  chan string
	OutputCh chan string
}

func NewAsk20Service() *Ask20Service {
	return &Ask20Service{
		Ask20Map: make(map[string][]*model.ChatCompletionMessage),
		InputCh:  make(chan string),
		OutputCh: make(chan string),
	}
}

func (s *Ask20Service) Ask20() {
	client := GetClient()
	ctx := context.Background()
	message := make([]*model.ChatCompletionMessage, 0)
	message = append(message, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleSystem,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String("你是一个专业的20个问题的主持人, 你先想好一个词语，然后需要给用户一个猜测的范围，比如三国人物，常见的动物，常见的植物等等。同时对用户的问题回答是或者否并统计用户提问次数。用户需要在20次问题内猜出答案。你来告诉用户他猜的对不对"),
		},
	})

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
			message = s.Ask20Polish(message)
		}
	}
}

func (s *Ask20Service) Ask20Polish(msg []*model.ChatCompletionMessage) []*model.ChatCompletionMessage {
	client := GetClient()

	sb := strings.Builder{}
	for i := 1; i < len(msg); i++ {
		if msg[i].Role == model.ChatMessageRoleUser {
			sb.WriteString(fmt.Sprintf("玩家说: %s \n", *msg[i].Content.StringValue))
		} else {
			sb.WriteString(fmt.Sprintf("主持人说: %s \n", *msg[i].Content.StringValue))
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
			StringValue: volcengine.String(fmt.Sprintf("你是一个专业的20个问题的主持人, 你先想好一个词语，然后需要给用户一个猜测的范围，比如三国人物，常见的动物，常见的植物等等。同时对用户的问题回答是或者否并统计用户提问次数。用户需要在20次问题内猜出答案。你来告诉用户他猜的对不对。背景提要: %s", *resp.Choices[0].Message.Content.StringValue)),
		},
	})
	return newMessages
}

func (s *Ask20Service) Ask20Http(name string, msg []*model.ChatCompletionMessage) error {
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

	s.Ask20Map[name] = msg

	return nil
}

func (s *Ask20Service) Ask20Handler(c *gin.Context) {
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
		delete(s.Ask20Map, req.Name)
		c.JSON(http.StatusOK, gin.H{"error": "重置成功"})
		return
	}
	if _, ok := s.Ask20Map[req.Name]; !ok {
		s.Ask20Map[req.Name] = make([]*model.ChatCompletionMessage, 0)
	}
	if len(s.Ask20Map[req.Name]) == 0 {
		s.Ask20Map[req.Name] = append(s.Ask20Map[req.Name], &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleSystem,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String("你是一个专业的20个问题的主持人, 你先想好一个词语，然后需要给用户一个猜测的范围，比如三国人物，常见的动物，常见的植物等等。同时对用户的问题回答是或者否并统计用户提问次数。用户需要在20次问题内猜出答案。你来告诉用户他猜的对不对"),
			},
		})
	}
	s.Ask20Map[req.Name] = append(s.Ask20Map[req.Name], &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleUser,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String(req.Content),
		},
	})

	err := s.Ask20Http(req.Name, s.Ask20Map[req.Name])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	value := *s.Ask20Map[req.Name][len(s.Ask20Map[req.Name])-1].Content.StringValue
	if len(s.Ask20Map[req.Name]) > 10 {
		s.Ask20Map[req.Name] = s.Ask20Polish(s.Ask20Map[req.Name])
	}
	c.JSON(http.StatusOK, gin.H{"success": value})
}

func (s *Ask20Service) Ask20Socketfunc(c *gin.Context) {
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
