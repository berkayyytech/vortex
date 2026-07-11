package files

import "os"

type FileList struct {
	Files []string
}

func GetAllFilesDirectories() FileList {

	entries, err := os.ReadDir(".")
	var files = []string{}
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return FileList{Files: files}
}

func (f FileList) FormatFiles() string {
	if len(f.Files) == 0 {
		return "No files found in current directory."
	}
	res := "📄 Current Directory Files:\n─────────────────────────────\n"
	for _, file := range f.Files {
		res += "  " + file + "\n"
	}
	return res
}
