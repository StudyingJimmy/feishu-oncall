// Package idgen 生成各类业务 ID（工单号、会话 ID、对象存储 key）。
package idgen

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 去掉易混字符

// TicketNo 生成工单号，形如 OC-20260914-7F3K。
// 说明：骨架阶段用「日期 + 随机码」保证可读性与低碰撞；接库后可换成数据库自增序列。
func TicketNo(prefix string, t time.Time) string {
	if prefix == "" {
		prefix = "OC"
	}
	return fmt.Sprintf("%s-%s-%s", strings.ToUpper(prefix), t.Format("20060102"), randomCode(4))
}

// SessionID 预检会话 ID，形如 ps-20260914T1030-8H2M。
func SessionID(t time.Time) string {
	return fmt.Sprintf("ps-%s-%s", t.Format("20060102T1504"), randomCode(4))
}

// RequestID 请求追踪 ID。
func RequestID() string {
	return fmt.Sprintf("req-%d-%s", time.Now().UnixMilli(), randomCode(6))
}

// ObjectKey 对象存储里的 key，按 日期/模块/随机 组织，方便按天归档与生命周期清理。
func ObjectKey(module string, ext string) string {
	now := time.Now()
	if module == "" {
		module = "misc"
	}
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("%s/%s/%s%s", module, now.Format("2006/01/02"), randomCode(10), ext)
}

func randomCode(n int) string {
	var sb strings.Builder
	max := big.NewInt(int64(len(codeAlphabet)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			sb.WriteByte(codeAlphabet[i%len(codeAlphabet)])
			continue
		}
		sb.WriteByte(codeAlphabet[idx.Int64()])
	}
	return sb.String()
}
