package db

import (
	"context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

var LocalDB *gorm.DB

func Setup() {
	InitLocalDB()
}

func InitLocalDB() {
	var err error
	// 数据库连接信息
	dsn := "root:@tcp(127.0.0.1:3306)/play_with_ds?charset=utf8mb4&parseTime=True&loc=Local"
	// 连接数据库
	LocalDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
		panic(err)
	}
}

func AddRoles(ctx context.Context, name, description string) (int, error) {
	newRole := &RoleDesign{
		Name:           name,
		Content:        description,
		CreateOperator: "system",
	}
	result := LocalDB.Model(&RoleDesign{}).Create(newRole)
	if result.Error != nil {
		return 0, result.Error
	}
	return newRole.ID, nil
}

func AddDNDStory(ctx context.Context, roleID int) (int, error) {
	story := &DndStory{
		RoleDesignID: roleID,
	}
	result := LocalDB.Model(&DndStory{}).Create(story)
	if result.Error != nil {
		return 0, result.Error
	}
	return story.ID, nil
}

func GetStoryDetailByID(ctx context.Context, storyID int) (*DndStory, error) {
	story := &DndStory{}
	result := LocalDB.Model(&DndStory{}).Preload("RoleDesign").Where("id = ?", storyID).First(story)
	if result.Error != nil {
		return nil, result.Error
	}
	return story, nil
}

func AddStoryDetail(ctx context.Context, storyID int, roleType, content, image string) error {
	detail := &StoryDetail{
		DndStoryID: storyID,
		Role:       roleType,
		Content:    content,
		Image:      image,
	}
	return LocalDB.Model(&StoryDetail{}).Create(detail).Error
}

func GetStoryDetailsByStoryID(ctx context.Context, storyID int) ([]*StoryDetail, error) {
	details := make([]*StoryDetail, 0)
	result := LocalDB.Model(&StoryDetail{}).Where("dnd_story_id = ?", storyID).Find(&details)
	if result.Error != nil {
		return nil, result.Error
	}
	return details, nil
}
