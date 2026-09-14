// Package es 封装 Elasticsearch：知识库切片索引与检索。
//
// 检索策略：当前是 BM25 关键词召回；向量召回（dense_vector + embedding 服务）
// 的接入点见 Search 里的 TODO，混合排序也在那里做。
package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"bokeoncall/internal/conf"
)

// chunk 索引默认分词器。装上 IK 插件后改成 ik_max_word（见 scripts/elasticsearch/Dockerfile.ik）。
const chunkAnalyzer = "standard"

// ChunkDoc 知识库切片文档。
type ChunkDoc struct {
	DocKey    string    `json:"doc_key"`
	DocID     uint64    `json:"doc_id"`
	ChunkNo   int       `json:"chunk_no"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	TeamKey   string    `json:"team_key,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	UpdatedAt string    `json:"updated_at"`
	Vector    []float32 `json:"vector,omitempty"` // TODO: 接入 embedding 后写入
}

// ChunkHit 检索命中。
type ChunkHit struct {
	DocKey  string  `json:"doc_key"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// Client ES 客户端。
type Client struct {
	es          *elasticsearch.Client
	indexChunks string
	indexTicket string
	timeout     time.Duration
}

// NewClient 创建客户端（不发起连接，首次请求时才连）。
func NewClient(cfg conf.ESConfig) (*Client, error) {
	timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	c, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 ES 客户端失败: %w", err)
	}
	return &Client{
		es:          c,
		indexChunks: cfg.IndexChunks,
		indexTicket: cfg.IndexTickets,
		timeout:     timeout,
	}, nil
}

// Ping 健康检查。
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("ES ping 失败: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("ES ping 返回错误: %s", res.String())
	}
	return nil
}

// EnsureIndices 确保索引存在（幂等）。
func (c *Client) EnsureIndices(ctx context.Context) error {
	return c.ensureChunkIndex(ctx)
}

func (c *Client) ensureChunkIndex(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	exists, err := c.es.Indices.Exists([]string{c.indexChunks}, c.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("检查索引失败: %w", err)
	}
	defer exists.Body.Close()
	if exists.StatusCode == 200 {
		return nil
	}
	if exists.StatusCode != 404 {
		return fmt.Errorf("检查索引返回异常状态: %d", exists.StatusCode)
	}

	mapping := map[string]any{
		"settings": map[string]any{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"doc_key":    map[string]any{"type": "keyword"},
				"doc_id":     map[string]any{"type": "long"},
				"chunk_no":   map[string]any{"type": "integer"},
				"title":      map[string]any{"type": "text", "analyzer": chunkAnalyzer},
				"content":    map[string]any{"type": "text", "analyzer": chunkAnalyzer},
				"team_key":   map[string]any{"type": "keyword"},
				"tags":       map[string]any{"type": "keyword"},
				"updated_at": map[string]any{"type": "date"},
			},
		},
	}
	body, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	res, err := c.es.Indices.Create(c.indexChunks, c.es.Indices.Create.WithBody(bytes.NewReader(body)), c.es.Indices.Create.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("创建索引返回错误: %s", res.String())
	}
	return nil
}

// IndexChunk 写入单个切片。
func (c *Client) IndexChunk(ctx context.Context, doc ChunkDoc) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("%s-%d", doc.DocKey, doc.ChunkNo)
	res, err := c.es.Index(c.indexChunks, bytes.NewReader(payload), c.es.Index.WithDocumentID(id), c.es.Index.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("写入切片失败: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("写入切片返回错误: %s", res.String())
	}
	return nil
}

// BulkIndexChunks 批量写入切片（NDJSON）。
func (c *Client) BulkIndexChunks(ctx context.Context, docs []ChunkDoc) error {
	if len(docs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout*4)
	defer cancel()

	var buf bytes.Buffer
	for _, doc := range docs {
		meta := map[string]any{"index": map[string]any{"_index": c.indexChunks, "_id": fmt.Sprintf("%s-%d", doc.DocKey, doc.ChunkNo)}}
		metaLine, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		docLine, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		buf.Write(metaLine)
		buf.WriteByte('\n')
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	res, err := c.es.Bulk(bytes.NewReader(buf.Bytes()), c.es.Bulk.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("批量写入失败: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("批量写入返回错误: %s", string(raw))
	}
	return nil
}

// DeleteChunksByDoc 删除某个文档的全部切片（重新导入时用）。
func (c *Client) DeleteChunksByDoc(ctx context.Context, docKey string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	query := map[string]any{"query": map[string]any{"term": map[string]any{"doc_key": docKey}}}
	body, err := json.Marshal(query)
	if err != nil {
		return err
	}
	res, err := c.es.DeleteByQuery([]string{c.indexChunks}, bytes.NewReader(body), c.es.DeleteByQuery.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("删除切片失败: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("删除切片返回错误: %s", res.String())
	}
	return nil
}

// Search 关键词检索。
//
// TODO(向量): 接入 embedding 服务后，这里改成 hybrid：
//  1. knn 查询 vector 字段召回 topK；
//  2. 与 BM25 结果做 RRF/加权融合；
//  3. 分数低于阈值时不建议自助，直接转人工（阈值放 conf.Precheck.MinScore）。
func (c *Client) Search(ctx context.Context, keyword string, topK int) ([]ChunkHit, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, nil
	}
	if topK <= 0 {
		topK = 5
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	query := map[string]any{
		"size":    topK,
		"_source": []string{"doc_key", "title", "content"},
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  keyword,
				"fields": []string{"title^2", "content"},
				"type":   "best_fields",
			},
		},
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req := esapi.SearchRequest{
		Index: []string{c.indexChunks},
		Body:  bytes.NewReader(body),
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, fmt.Errorf("检索失败: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		raw, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("检索返回错误: %s", string(raw))
	}

	var parsed struct {
		Hits struct {
			Hits []struct {
				Score  float64  `json:"_score"`
				Source ChunkDoc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("解析检索结果失败: %w", err)
	}

	hits := make([]ChunkHit, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		hits = append(hits, ChunkHit{
			DocKey:  h.Source.DocKey,
			Title:   h.Source.Title,
			Content: h.Source.Content,
			Score:   h.Score,
		})
	}
	return hits, nil
}
