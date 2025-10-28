package test

import (
	"fmt"
	"os"

	"github.com/labstack/gommon/random"
)

func RandHex(n uint8) string {
	r := random.New()
	return r.String(n, random.Hex)
}

func PrepareContentFolder() (func(), string, error) {
	// generate random 8 hex chars for tmp dir
	chars := RandHex(8)
	dirName, err := os.MkdirTemp("", fmt.Sprintf("oauth-static-webserver-%s", chars))
	if err != nil {
		return nil, "", err
	}
	// create 3 page dirs with file.txt (content the counter 1 to 3)
	for i := 0; i < 3; i++ {
		pageDir := fmt.Sprintf("%s/page%d", dirName, i+1)
		err := os.Mkdir(pageDir, 0o755)
		if err != nil {
			_ = os.RemoveAll(dirName)
			return nil, "", err
		}
		f, err := os.Create(fmt.Sprintf("%s/file.txt", pageDir))
		if err != nil {
			_ = os.RemoveAll(dirName)
			return nil, "", err
		}
		_, err = f.WriteString(fmt.Sprintf("page=%d", i+1))
		if err != nil {
			_ = os.RemoveAll(dirName)
			return nil, "", err
		}
		_ = f.Close()
	}
	return func() {
		_ = os.RemoveAll(dirName)
	}, dirName, nil
}
