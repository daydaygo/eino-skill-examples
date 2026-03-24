package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SearchInput struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索查询内容"`
	TopK  int    `json:"top_k" jsonschema_description:"返回结果数量，默认3"`
}

func NewSearchTool(store *MemoryStore) (tool.BaseTool, error) {
	return utils.InferTool(
		"search_knowledge",
		"搜索知识库获取相关信息",
		func(ctx context.Context, input *SearchInput) (string, error) {
			if input.TopK <= 0 {
				input.TopK = 3
			}

			docs := store.Search(ctx, input.Query, input.TopK)
			if len(docs) == 0 {
				return "未找到相关内容。", nil
			}

			var sb strings.Builder
			sb.WriteString("找到以下相关内容：\n\n")
			for i, doc := range docs {
				sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, doc.Content))
			}

			return sb.String(), nil
		},
	)
}
