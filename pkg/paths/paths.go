package paths

import (
	"os"
	"path/filepath"
)

// ExeDir 返回可执行文件所在目录（开发时为当前工作目录）
func ExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exe)
}

// UserDir 返回用户数据目录：{exeDir}/{qq}/
func UserDir(qq string) string {
	return filepath.Join(ExeDir(), qq)
}

// EnsureUserDir 创建用户目录
func EnsureUserDir(qq string) (string, error) {
	dir := UserDir(qq)
	return dir, os.MkdirAll(dir, 0755)
}

// UserDBPath 用户会话数据库
func UserDBPath(qq string) string {
	return filepath.Join(UserDir(qq), "app.db")
}

// ExportJSONPath 主导出文件
func ExportJSONPath(qq string) string {
	return filepath.Join(UserDir(qq), qq+"_export.json")
}

// ActivitiesJSONPath 活动记录
func ActivitiesJSONPath(qq string) string {
	return filepath.Join(UserDir(qq), qq+"_activities.json")
}

// ViewerHTMLPath 浏览页
func ViewerHTMLPath(qq string) string {
	return filepath.Join(UserDir(qq), qq+"_view.html")
}
