package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

var inputCh = make(chan string)
var outputCh = make(chan string)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有跨域请求
		return true
	},
}

type Ask20Request struct {
	Auth    string `json:"auth"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

var Ask20Map = make(map[string][]*model.ChatCompletionMessage)
var NormalMap = make(map[string][]*model.ChatCompletionMessage)

func main() {
	// 创建 Gin 实例
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:44445"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// WebSocket 路由
	r.GET("/ws", func(c *gin.Context) {
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
			inputCh <- string(message)
			output := <-outputCh

			// 将消息原样返回给客户端
			if err := conn.WriteMessage(messageType, []byte(strings.TrimSpace(output))); err != nil {
				log.Println("发送消息失败:", err)
				break
			}
		}

		log.Println("客户端已断开连接")
	})

	go Ask20()
	// 启动 Gin 服务器

	Ask20Map = make(map[string][]*model.ChatCompletionMessage)
	NormalMap = make(map[string][]*model.ChatCompletionMessage)
	r.POST("/ask20", func(c *gin.Context) {
		req := &Ask20Request{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Auth != "zzyztyy" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "auth error"})
			return
		}
		if req.Content == "重置" {
			delete(Ask20Map, req.Name)
			c.JSON(http.StatusOK, gin.H{"error": "重置成功"})
			return
		}
		if _, ok := Ask20Map[req.Name]; !ok {
			Ask20Map[req.Name] = make([]*model.ChatCompletionMessage, 0)
		}
		if len(Ask20Map[req.Name]) == 0 {
			Ask20Map[req.Name] = append(Ask20Map[req.Name], &model.ChatCompletionMessage{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String("你是一个专业的20个问题的主持人, 你先想好一个词语，然后需要给用户一个猜测的范围，比如三国人物，常见的动物，常见的植物等等。同时对用户的问题回答是或者否并统计用户提问次数。用户需要在20次问题内猜出答案。你来告诉用户他猜的对不对"),
				},
			})
		}
		Ask20Map[req.Name] = append(Ask20Map[req.Name], &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(req.Content),
			},
		})

		err := Ask20Http(req.Name, Ask20Map[req.Name])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		value := *Ask20Map[req.Name][len(Ask20Map[req.Name])-1].Content.StringValue
		if len(Ask20Map[req.Name]) > 10 {
			Ask20Map[req.Name] = Ask20Polish(Ask20Map[req.Name])
		}
		c.JSON(http.StatusOK, gin.H{"success": value})
	})
	r.POST("/ds", func(c *gin.Context) {
		req := &Ask20Request{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Auth != "zzyztyy" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "auth error"})
			return
		}
		if req.Content == "重置" {
			delete(NormalMap, req.Name)
			c.JSON(http.StatusOK, gin.H{"error": "重置成功"})
			return
		}
		if _, ok := NormalMap[req.Name]; !ok {
			NormalMap[req.Name] = make([]*model.ChatCompletionMessage, 0)
		}
		if len(NormalMap[req.Name]) == 0 {
			NormalMap[req.Name] = append(NormalMap[req.Name], &model.ChatCompletionMessage{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String("你是deepseek-v3, 强大的AI，请帮忙解决人类的问题"),
				},
			})
		}
		NormalMap[req.Name] = append(NormalMap[req.Name], &model.ChatCompletionMessage{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(req.Content),
			},
		})

		err := AskNormal(req.Name, NormalMap[req.Name])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		value := *NormalMap[req.Name][len(NormalMap[req.Name])-1].Content.StringValue
		if len(NormalMap[req.Name]) > 10 {
			NormalMap[req.Name] = NormalPolish(NormalMap[req.Name])
		}

		c.JSON(http.StatusOK, gin.H{"success": value})
	})

	if err := r.Run(":44444"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}

func DND() {
	client := arkruntime.NewClientWithApiKey(
		os.Getenv("ARK_API_KEY"),
	)
	ctx := context.Background()
	message := make([]*model.ChatCompletionMessage, 0)
	message = append(message, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleSystem,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String("你是一个专业的DND的DM, 你的回答应该尽量保持DM的风格，不需要太多关于背景的描述，只需要作为一个DM进行流程的引导。你的回答应该尽量简洁。"),
		},
	})
	//reader := bufio.NewReader(os.Stdin)

	for {
		// 获取用户的输入
		//input, err := reader.ReadString('\n')
		//if err != nil {
		//	break
		//}
		//input = strings.TrimSpace(input)
		//if input == "q!" {
		//	break
		//}

		input := <-inputCh

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
		outputCh <- strings.Join(reply, "")

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
			message = DNDPolish(message)
		}
	}
}

func DNDPolish(msg []*model.ChatCompletionMessage) []*model.ChatCompletionMessage {
	client := arkruntime.NewClientWithApiKey(
		//os.Getenv("ARK_API_KEY"),
		"d0797c1d-dc09-47af-9e9d-d40079ffd006",
	)

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
			StringValue: volcengine.String(fmt.Sprintf("你是一个专业的DND的DM, 你的回答应该尽量保持DM的风格，不需要太多关于背景的描述，只需要作为一个DM进行流程的引导。你的回答应该尽量简洁。背景提要: %s", *resp.Choices[0].Message.Content.StringValue)),
		},
	})
	return newMessages
}

func Ask20() {
	client := arkruntime.NewClientWithApiKey(
		//os.Getenv("ARK_API_KEY"),
		"d0797c1d-dc09-47af-9e9d-d40079ffd006",
	)
	ctx := context.Background()
	message := make([]*model.ChatCompletionMessage, 0)
	message = append(message, &model.ChatCompletionMessage{
		Role: model.ChatMessageRoleSystem,
		Content: &model.ChatCompletionMessageContent{
			StringValue: volcengine.String("你是一个专业的20个问题的主持人, 你先想好一个词语，然后需要给用户一个猜测的范围，比如三国人物，常见的动物，常见的植物等等。同时对用户的问题回答是或者否并统计用户提问次数。用户需要在20次问题内猜出答案。你来告诉用户他猜的对不对"),
		},
	})
	//reader := bufio.NewReader(os.Stdin)

	for {
		// 获取用户的输入
		//input, err := reader.ReadString('\n')
		//if err != nil {
		//	break
		//}
		//input = strings.TrimSpace(input)
		//if input == "q!" {
		//	break
		//}

		input := <-inputCh

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
		outputCh <- strings.Join(reply, "")

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
			message = Ask20Polish(message)
		}
	}
}

func Ask20Polish(msg []*model.ChatCompletionMessage) []*model.ChatCompletionMessage {
	client := arkruntime.NewClientWithApiKey(
		//os.Getenv("ARK_API_KEY"),
		"d0797c1d-dc09-47af-9e9d-d40079ffd006",
	)

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

func Ask20Http(name string, msg []*model.ChatCompletionMessage) error {
	client := arkruntime.NewClientWithApiKey(
		//os.Getenv("ARK_API_KEY"),
		"d0797c1d-dc09-47af-9e9d-d40079ffd006",
	)
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

	Ask20Map[name] = msg

	return nil
}

func AskNormal(name string, msg []*model.ChatCompletionMessage) error {
	client := arkruntime.NewClientWithApiKey(
		//os.Getenv("ARK_API_KEY"),
		"d0797c1d-dc09-47af-9e9d-d40079ffd006",
	)
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
	NormalMap[name] = msg

	return nil
}

func NormalPolish(msg []*model.ChatCompletionMessage) []*model.ChatCompletionMessage {
	client := arkruntime.NewClientWithApiKey(
		//os.Getenv("ARK_API_KEY"),
		"d0797c1d-dc09-47af-9e9d-d40079ffd006",
	)

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
