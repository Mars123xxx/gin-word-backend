package global

import (
	"gorm.io/gorm"
	"time"
)

// Article represents the article table in GORM
type Article struct {
	gorm.Model
	Title      string `gorm:"size:200"`
	Body       string `gorm:"type:text"`
	Source     string `gorm:"size:1000"`
	SourceHref string `gorm:"size:1000"`
	Headpic    string `gorm:"size:1000"`
	Abstract   string `gorm:"type:text"`
}

// Word represents the word table in GORM
type Word struct {
	gorm.Model
	Word         string `gorm:"size:80;unique;not null"`
	Language     string `gorm:"size:20;default:'MY'"`
	Sentences    []Sentence
	RootWord     string `gorm:"size:80"`
	RootMeaning  string `gorm:"size:80"`
	Meanings     []Meaning
	Collocations []Collocation
	Relatives    []Relative
	Looklikes    []Looklike
	Synonyms     []*Word `gorm:"many2many:synonyms;"`
	StudyLogs    []StudyLog
}

// Sentence represents the sentence table in GORM
type Sentence struct {
	gorm.Model
	Content string `gorm:"size:200;not null"`
	WordID  uint   `gorm:"not null"`
}

// Collocation represents the collocation table in GORM
type Collocation struct {
	gorm.Model
	Content string `gorm:"size:200;not null"`
	WordID  uint   `gorm:"not null"`
}

// Relative represents the relative table in GORM
type Relative struct {
	gorm.Model
	Content string `gorm:"size:200;not null"`
	WordID  uint   `gorm:"not null"`
}

// Looklike represents the looklike table in GORM
type Looklike struct {
	gorm.Model
	Word   string `gorm:"size:200;not null"`
	WordID uint   `gorm:"not null"`
}

// Meaning represents the meaning table in GORM
type Meaning struct {
	gorm.Model
	PartOfSpeech string `gorm:"size:20;not null"`
	Definition   string `gorm:"size:200;not null"`
	WordID       uint   `gorm:"not null"`
}

// User represents the user table in GORM
type User struct {
	gorm.Model
	Username  string `gorm:"size:80;unique;not null"`
	Phone     string `gorm:"size:20;unique;not null"`
	Avatar    string `gorm:"size:200;default:'https://wechat-1317403776.cos.ap-beijing.myqcloud.com/%E5%9B%BE%E7%89%87/R-C.jpg'"`
	Schedule  int    `gorm:"default:20"`
	StudyLogs []StudyLog
}

// StudyLog represents the study_log table in GORM
type StudyLog struct {
	gorm.Model
	UserID         uint      `gorm:"not null"`
	WordID         uint      `gorm:"not null"`
	StartStudyTime time.Time `gorm:"default:current_timestamp"`
	StudyTimes     int       `gorm:"default:0"`
	NextStudyTime  *time.Time
}
