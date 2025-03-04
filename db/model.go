package db

import _ "gorm.io/gorm"

// RoleDesign 角色设计表
type RoleDesign struct {
	ID             int    `gorm:"primaryKey;autoIncrement"`
	Name           string `gorm:"type:varchar(127);not null"`
	Content        string `gorm:"type:varchar(1023);not null"`
	CreateOperator string `gorm:"type:varchar(255);not null"`
}

// TableName 设置 RoleDesign 对应的表名
func (RoleDesign) TableName() string {
	return "role_design" // 指定表名
}

// DndStory DND 故事表
type DndStory struct {
	ID           int        `gorm:"primaryKey;autoIncrement"`
	RoleDesignID int        `gorm:"not null;index"`
	RoleDesign   RoleDesign `gorm:"foreignKey:RoleDesignID;references:ID"`
}

// TableName 设置 RoleDesign 对应的表名
func (DndStory) TableName() string {
	return "dnd_story" // 指定表名
}

// StoryDetail 故事详情表
type StoryDetail struct {
	ID         int      `gorm:"primaryKey;autoIncrement"`
	DndStoryID int      `gorm:"not null;index"`
	Role       string   `gorm:"type:varchar(127);not null;comment:'player | dm'"`
	Content    string   `gorm:"type:text;not null"`
	Image      string   `gorm:"type:varchar(255);not null;default:''"`
	DndStory   DndStory `gorm:"foreignKey:DndStoryID;references:ID"`
}

// TableName 设置 RoleDesign 对应的表名
func (StoryDetail) TableName() string {
	return "story_detail" // 指定表名
}
