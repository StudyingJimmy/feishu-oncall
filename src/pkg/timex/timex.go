// Package timex 时间工具：统一时区与常用格式化（存储用 UTC，展示用东八区）。
package timex

import (
	"strconv"
	"time"
)

// cst 东八区。
var cst = time.FixedZone("CST", 8*3600)

// Now 当前 UTC 时间（数据库统一存 UTC）。
func Now() time.Time { return time.Now().UTC() }

// NowCN 当前东八区时间（展示、定时任务判断用）。
func NowCN() time.Time { return time.Now().In(cst) }

// FormatCN 东八区可读格式，如 2026-09-14 18:20:31。
func FormatCN(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.In(cst).Format("2006-01-02 15:04:05")
}

// Humanize 粗略的相对时间（卡片文案用）。
func Humanize(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + " 分钟前"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + " 小时前"
	default:
		return strconv.Itoa(int(d.Hours()/24)) + " 天前"
	}
}

// AddMinutes 时间加分钟。
func AddMinutes(t time.Time, n int) time.Time { return t.Add(time.Duration(n) * time.Minute) }
