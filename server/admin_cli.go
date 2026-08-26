// 管理员密码命令行管理。
//
// 忘记密码场景：
//
//	1. 停止服务（Ctrl+C）
//	2. 带重置参数启动：./fuwari-server -re pwd
//	   交互输入新密码两次（掩码显示）→ 校验一致性/长度 → bcrypt 写入数据库
//	   → 提示「✅ 管理员密码已重置，请重新启动服务」→ 按任意键退出
//	3. 正常启动：./fuwari-server
//
// 首次启动（数据库无密码哈希）时由 ensureAdminPassword 引导：
//   - 配置了 ADMIN_TOKEN → 作为初始密码写入数据库（旧部署无缝迁移）；
//   - 未配置 → 生成随机密码并打印到启动日志（仅显示一次）。
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"fuwari-server/config"
	"fuwari-server/models"

	"golang.org/x/term"
)

// isResetPwdMode 判断命令行是否携带密码重置参数：
// ./fuwari-server -re pwd | --re pwd | -re=pwd
func isResetPwdMode() bool {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := strings.TrimLeft(args[i], "-")
		switch {
		case a == "re" && i+1 < len(args) && strings.ToLower(strings.TrimLeft(args[i+1], "-")) == "pwd":
			return true
		case strings.HasPrefix(a, "re=") && strings.EqualFold(strings.TrimPrefix(a, "re="), "pwd"):
			return true
		}
	}
	return false
}

// runResetPassword 交互式重置管理员密码（忘记密码场景）。
// 流程：输入新密码两次（掩码）→ 校验一致性/长度 → bcrypt 入库 → 提示重启。
func runResetPassword() {
	if err := models.InitDatabase(); err != nil {
		fmt.Printf("❌ 数据库初始化失败: %v\n", err)
		waitAnyKey()
		os.Exit(1)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		p1 := readPassword("请输入新密码: ")
		p2 := readPassword("请再次输入: ")
		if p1 != p2 {
			fmt.Println("❌ 两次输入不一致，请重试")
			continue
		}
		if len(p1) < 6 {
			fmt.Println("❌ 密码至少 6 个字符，请重试")
			continue
		}
		hash, err := models.HashPassword(p1)
		if err != nil {
			fmt.Printf("❌ 密码加密失败: %v\n", err)
			waitAnyKey()
			os.Exit(1)
		}
		if err := models.SetAdminPasswordHash(hash); err != nil {
			fmt.Printf("❌ 保存密码失败: %v\n", err)
			waitAnyKey()
			os.Exit(1)
		}
		fmt.Println("✅ 管理员密码已重置，请重新启动服务")
		waitAnyKey()
		os.Exit(0)
	}
	fmt.Println("❌ 多次输入无效，已放弃重置")
	waitAnyKey()
	os.Exit(1)
}

// ensureAdminPassword 首次启动时引导初始管理员密码（数据库无密码哈希时）。
func ensureAdminPassword() {
	if models.HasAdminPassword() {
		return
	}
	initial := config.AdminToken
	if initial == "" {
		initial = randomPassword(12)
		log.Println("========================================")
		log.Printf("  首次启动已生成随机管理员密码（仅显示一次）: %s", initial)
		log.Println("  修改方式: /editor 页面（知道密码）或 ./fuwari-server -re pwd（忘记密码）")
		log.Println("========================================")
	} else {
		log.Println("首次启动：已将 ADMIN_TOKEN 作为初始管理员密码写入数据库")
	}
	hash, err := models.HashPassword(initial)
	if err != nil {
		log.Fatalf("初始密码加密失败: %v", err)
	}
	if err := models.SetAdminPasswordHash(hash); err != nil {
		log.Fatalf("初始密码保存失败: %v", err)
	}
}

// stdinReader 非终端（管道/自动化）输入共享 reader：
// 每次新建 bufio.Reader 会因内部预读缓冲吞掉后续输入，必须全程复用。
var stdinReader = bufio.NewReader(os.Stdin)

// readPassword 读取密码：终端下掩码输入；非终端（管道/CI/自动化）降级为明文读取。
func readPassword(prompt string) string {
	fmt.Print(prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err == nil {
			return string(b)
		}
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimSpace(line)
}

// waitAnyKey 等待用户按键后退出（Windows pause 语义；非终端下直接返回）
func waitAnyKey() {
	fmt.Println("按任意键继续...")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		var d string
		_, _ = fmt.Scanln(&d)
	} else {
		_, _ = stdinReader.ReadString('\n')
	}
}

// randomPassword 生成 n 字节十六进制随机密码（仅用于首次启动引导）
func randomPassword(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "change-me-" + strconv.Itoa(os.Getpid())
	}
	return hex.EncodeToString(b)
}
