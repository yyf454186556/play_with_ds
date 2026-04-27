package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"play_with_ds/db"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

type DNDService struct {
	DNDMap   map[int][]*model.ChatCompletionMessage
	InputCh  chan string
	OutputCh chan string
	FaceMap  map[int]string
}

func NewDNDService() *DNDService {
	return &DNDService{
		DNDMap:   make(map[int][]*model.ChatCompletionMessage),
		InputCh:  make(chan string),
		OutputCh: make(chan string),
		FaceMap:  make(map[int]string),
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
	if req.StoryID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request. invalid story id"})
		return
	}
	if req.Content == "重置" {
		delete(s.DNDMap, req.StoryID)
		delete(s.FaceMap, req.StoryID)
		c.JSON(http.StatusOK, gin.H{"error": "重置成功"})
		return
	}

	story, err := db.GetStoryDetailByID(c, req.StoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "call db error"})
		return
	}

	s.FaceMap[req.StoryID] = story.RoleDesign.Content

	details, err := db.GetStoryDetailsByStoryID(c, req.StoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "call db error"})
		return
	}
	if _, ok := s.DNDMap[req.StoryID]; !ok || len(details) == 0 {
		s.DNDMap[req.StoryID] = make([]*model.ChatCompletionMessage, 0)
	}

	if len(s.DNDMap[req.StoryID]) == 0 {
		s.DNDMap[req.StoryID] = append(s.DNDMap[req.StoryID], &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleSystem,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(dnd_system_description),
			},
		})
	} else {
		for _, d := range details {
			if d.Role == "player" {
				s.DNDMap[req.StoryID] = append(s.DNDMap[req.StoryID], &model.ChatCompletionMessage{
					Role: model.ChatMessageRoleUser,
					Content: &model.ChatCompletionMessageContent{
						StringValue: volcengine.String(d.Content),
					},
				})
			} else {
				s.DNDMap[req.StoryID] = append(s.DNDMap[req.StoryID], &model.ChatCompletionMessage{
					Role: model.ChatMessageRoleAssistant,
					Content: &model.ChatCompletionMessageContent{
						StringValue: volcengine.String(d.Content),
					},
				})
			}
		}

		s.DNDMap[req.StoryID] = append(s.DNDMap[req.StoryID], &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(req.Content),
			},
		})
	}

	err = s.AskDND(req.StoryID, s.DNDMap[req.StoryID])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 写用户的输入
	err = db.AddStoryDetail(c, req.StoryID, "player", req.Content, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	uuid := ""
	value := *s.DNDMap[req.StoryID][len(s.DNDMap[req.StoryID])-1].Content.StringValue
	if len(s.DNDMap[req.StoryID]) > 8 {
		s.DNDMap[req.StoryID] = s.DNDPolish(s.DNDMap[req.StoryID])
		uuid = GetPicture(value, s.FaceMap[req.StoryID])
		// 写dm的回复
		err = db.AddStoryDetail(c, req.StoryID, "dm", value, uuid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// 写dm的回复
		err = db.AddStoryDetail(c, req.StoryID, "dm", value, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": value, "uuid": uuid})
}

func (s *DNDService) AskDND(id int, msg []*model.ChatCompletionMessage) error {
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

	s.DNDMap[id] = msg

	return nil
}

func (s *DNDService) DNDAddRole(c *gin.Context) {
	req := &AddRoleRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Auth != "zzyztyy" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth error"})
		return
	}

	roleID, err := db.AddRoles(c, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db create role error"})
		return
	}
	storyID, err := db.AddDNDStory(c, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db create story error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": "ok", "story_id": storyID})
}
