package dto

// KbHit 知识库命中项（预检卡片里展示的检索结果）。
type KbHit struct {
	DocKey  string  `json:"doc_key"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}
