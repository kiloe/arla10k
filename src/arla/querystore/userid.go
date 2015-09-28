package querystore

import (
	"os/exec"
	"strconv"
	"strings"
)

// Use "id" command to get Uid/Gid to avoid need for cgo
func getSysUserId(flag, username string) (id uint32, err error) {
	exe, err := exec.LookPath("id")
	if err != nil {
		return
	}
	args := []string{flag, username}
	cmd := exec.Command(exe, args...)
	res, err := cmd.Output()
	if err != nil {
		return
	}
	n := strings.TrimSpace(string(res))
	id64, err := strconv.ParseUint(n, 10, 32)
	if err != nil {
		return
	}
	return uint32(id64), nil
}

func getUid(username string) (id uint32, err error) {
	return getSysUserId("-u", username)
}

func getGid(username string) (id uint32, err error) {
	return getSysUserId("-u", username)
}
