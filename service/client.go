package service

import (
	"os"
	"sync"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
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
