package service

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"

	"github.com/volcengine/volc-sdk-golang/service/visual"
)

var client *arkruntime.Client

var once sync.Once

func GetClient() *arkruntime.Client {
	once.Do(func() {
		client = arkruntime.NewClientWithApiKey(
			os.Getenv("ARK_API_KEY"),
		)
	})
	return client
}

func GetPicture(prompt string) string {
	testAk := os.Getenv("ARK_API_VISION_AK_KEY")
	testSk := os.Getenv("ARK_API_VISION_KEY")

	visual.DefaultInstance.Client.SetAccessKey(testAk)
	visual.DefaultInstance.Client.SetSecretKey(testSk)

	reqBody := map[string]interface{}{
		"req_key": "high_aes_general_v21_L",
		"prompt":  fmt.Sprintf("漫画风格，背景描述是这样的: %s。请根据文本生成图片", prompt),
		"width":   256,
		"height":  256,
	}
	resp, status, err := visual.DefaultInstance.CVProcess(reqBody)
	fmt.Println(status, err)
	binary_data_base64arr := make([]string, 0)

	xxx, ok := resp["data"].(map[string]interface{})
	if !ok {
		fmt.Print("can't change 0")
		return ""
	}

	slice, ok := xxx["binary_data_base64"].([]interface{})
	if !ok {
		fmt.Println("can't change")
		return ""
	}

	for _, v := range slice {
		str, ok := v.(string)
		if !ok {
			fmt.Println("Can't change 2.0")
			return ""
		}
		binary_data_base64arr = append(binary_data_base64arr, str)
	}

	decoded, err := base64.StdEncoding.DecodeString(binary_data_base64arr[0])
	if err != nil {
		panic(err)
	}

	uuid := uuid.NewString()

	err = os.WriteFile(fmt.Sprintf("./play_with_ds_front/public/%s.png", uuid), decoded, 0644)
	if err != nil {
		panic(err)
	}
	return uuid
}
