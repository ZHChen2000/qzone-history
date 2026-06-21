package paths

import (
	"os"
	"path/filepath"
)

func ExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exe)
}

func UserDir(qq string) string {
	return filepath.Join(ExeDir(), qq)
}

func EnsureUserDir(qq string) (string, error) {
	dir := UserDir(qq)
	return dir, os.MkdirAll(dir, 0755)
}

func UserDBPath(qq string) string {
	return filepath.Join(UserDir(qq), "app.db")
}

func ExportJSONPath(qq string) string {
	return filepath.Join(UserDir(qq), qq+"_export.json")
}

func ActivitiesJSONPath(qq string) string {
	return filepath.Join(UserDir(qq), qq+"_activities.json")
}

func ViewerHTMLPath(qq string) string {
	return filepath.Join(UserDir(qq), qq+"_view.html")
}
