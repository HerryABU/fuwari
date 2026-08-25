// Package version 集中管理 Fuwari 服务端版本号，供 handlers / main / 前端统一引用，
// 避免各文件硬编码版本不一致。
// 可通过构建参数覆盖：go build -ldflags "-X fuwari-server/version.AppVersion=v0.2.0"
package version

// AppVersion 当前版本号（不带 v 前缀，展示时自行拼接）。
var AppVersion = "0.1.0"
