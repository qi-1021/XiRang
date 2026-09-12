package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// skills 列表条目
type skillListEntry struct {
	Name        string `json:"name"`
	Pattern     string `json:"pattern"`
	ScriptPath  string `json:"script_path"`
	Description string `json:"description"`
	ScriptOK    bool   `json:"script_ok"`
}

func loadAllSkills() []skillListEntry {
	files, err := os.ReadDir(scriptsDir)
	if err != nil {
		return nil
	}
	var out []skillListEntry
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".meta.json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(scriptsDir, f.Name()))
		if err != nil {
			continue
		}
		var sk DiscoveredSkill
		if json.Unmarshal(data, &sk) != nil {
			continue
		}
		_, statErr := os.Stat(sk.ScriptPath)
		out = append(out, skillListEntry{
			Name:        sk.Name,
			Pattern:     sk.Pattern,
			ScriptPath:  sk.ScriptPath,
			Description: sk.Description,
			ScriptOK:    statErr == nil,
		})
	}
	return out
}

func runSkillsCLI(args []string) {
	// args: 子命令... 兼容 -skills 无参=list
	sub := "list"
	if len(args) > 0 {
		sub = args[0]
		args = args[1:]
	}

	switch sub {
	case "list", "":
		entries := loadAllSkills()
		fmt.Println("================================================================")
		fmt.Println("📜 息壤技能库 (.xirang/scripts/)")
		fmt.Println("================================================================")
		if len(entries) == 0 {
			fmt.Println("（空）使用过程中写入 .xirang/scripts/ 会自动沉淀指纹")
			return
		}
		for i, e := range entries {
			status := "OK"
			if !e.ScriptOK {
				status = "SCRIPT_MISSING"
			}
			fmt.Printf("%2d. [%s] %s\n", i+1, status, e.Name)
			fmt.Printf("    pattern: %s\n", e.Pattern)
			fmt.Printf("    script:  %s\n", e.ScriptPath)
			fmt.Printf("    desc:    %s\n", e.Description)
		}
		fmt.Println("\n子命令: list | show <name> | disable <name> | enable <name> | export <path> | import <path>")
		fmt.Println("示例: xirang -skills show FixPort")

	case "show":
		if len(args) < 1 {
			fmt.Println("用法: -skills show <name>")
			return
		}
		for _, e := range loadAllSkills() {
			if strings.EqualFold(e.Name, args[0]) || strings.EqualFold(filepath.Base(e.ScriptPath), args[0]) {
				b, _ := json.MarshalIndent(e, "", "  ")
				fmt.Println(string(b))
				return
			}
		}
		fmt.Println("未找到技能:", args[0])

	case "disable", "enable":
		if len(args) < 1 {
			fmt.Printf("用法: -skills %s <name>\n", sub)
			return
		}
		disabledDir := filepath.Join(scriptsDir, "disabled")
		_ = os.MkdirAll(disabledDir, 0755)
		var target *skillListEntry
		all := loadAllSkills()
		for i := range all {
			if strings.EqualFold(all[i].Name, args[0]) {
				e := all[i]
				target = &e
				break
			}
		}
		if target == nil {
			fmt.Println("未找到技能:", args[0])
			return
		}
		metaName := filepath.Base(target.ScriptPath) + ".meta.json"
		metaPath := filepath.Join(scriptsDir, metaName)
		if sub == "disable" {
			_ = os.Rename(metaPath, filepath.Join(disabledDir, metaName))
			if _, err := os.Stat(target.ScriptPath); err == nil {
				_ = os.Rename(target.ScriptPath, filepath.Join(disabledDir, filepath.Base(target.ScriptPath)))
			}
			fmt.Println("已禁用:", target.Name)
		} else {
			_ = os.Rename(filepath.Join(disabledDir, metaName), metaPath)
			fmt.Println("已启用:", target.Name)
		}

	case "export":
		if len(args) < 1 {
			fmt.Println("用法: -skills export <输出目录>")
			return
		}
		outDir := args[0]
		_ = os.MkdirAll(outDir, 0755)
		n := 0
		for _, e := range loadAllSkills() {
			data, err := os.ReadFile(filepath.Join(scriptsDir, filepath.Base(e.ScriptPath)+".meta.json"))
			if err != nil {
				continue
			}
			_ = os.WriteFile(filepath.Join(outDir, filepath.Base(e.ScriptPath)+".meta.json"), data, 0644)
			if b, err := os.ReadFile(e.ScriptPath); err == nil {
				_ = os.WriteFile(filepath.Join(outDir, filepath.Base(e.ScriptPath)), b, 0755)
			}
			n++
		}
		fmt.Printf("已导出 %d 个技能到 %s\n", n, outDir)

	case "import":
		if len(args) < 1 {
			fmt.Println("用法: -skills import <目录>")
			return
		}
		entries, _ := os.ReadDir(args[0])
		n := 0
		for _, f := range entries {
			if f.IsDir() {
				continue
			}
			src := filepath.Join(args[0], f.Name())
			dst := filepath.Join(scriptsDir, f.Name())
			b, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			perm := os.FileMode(0644)
			if !strings.HasSuffix(f.Name(), ".meta.json") {
				perm = 0755
			}
			_ = os.WriteFile(dst, b, perm)
			n++
		}
		fmt.Printf("已导入 %d 个文件到 %s\n", n, scriptsDir)

	default:
		fmt.Println("未知技能子命令:", sub)
		fmt.Println("可用: list show disable enable export import")
	}
}

// promptYes 简单确认
func promptYes(reader *bufio.Reader, msg string) bool {
	fmt.Print(msg)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	return line == "" || strings.EqualFold(line, "y") || strings.EqualFold(line, "yes")
}
