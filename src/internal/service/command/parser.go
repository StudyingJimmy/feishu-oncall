// 命令解析：路由判断与命令处理共用。
package command

import "strings"

// Command 群内命令。
type Command string

const (
	Upgrade  Command = "upgrade"  // /upgrade [层级] [原因]  升级
	Transfer Command = "transfer" // /transfer <团队|人> [原因]  转派
	Event    Command = "event"    // /event [说明]  升级故障作战室
	Solve    Command = "solve"    // /solve [结论]  标记已解决
	Save     Command = "save"     // /save [说明]  暂时挂起
)

// Usage 命令说明（帮助文案用）。
var Usage = map[Command]string{
	Upgrade:  "升级工单到上一层值班，如 /upgrade L2 网络抖动",
	Transfer: "转派给其它团队或同学，如 /transfer kb 值班",
	Event:    "升级为故障并拉起故障作战室，如 /event 用户无法登录",
	Solve:    "标记已解决，如 /solve 重试后恢复正常",
	Save:     "暂时挂起工单（保存现场，后续再恢复），如 /save 等上游修复",
}

// Parse 解析命令与参数：首个 token 必须是 /xxx，其余作为参数。
//
//	"@机器人 /upgrade L2 网络抖动" -> (Upgrade, "L2 网络抖动", true)
//	"@机器人 帮我看下"            -> ("", "", false)
func Parse(text string) (Command, string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", "", false
	}

	head, args := text, ""
	if idx := strings.IndexAny(text, " \t\n"); idx >= 0 {
		head = text[:idx]
		args = strings.TrimSpace(text[idx+1:])
	}

	cmd := Command(strings.ToLower(strings.TrimPrefix(head, "/")))
	switch cmd {
	case Upgrade, Transfer, Event, Solve, Save:
		return cmd, args, true
	default:
		return "", "", false
	}
}
