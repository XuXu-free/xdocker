package container

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if len(cmdArray) == 0 {
		return fmt.Errorf("run container get user command error, cmdArray is nil")
	}

	setUpMount()

	// find the absolute path of assigned cmd in cmdArray[0] by searching $PATH
	// which store the filename of the script or binary
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		log.Errorf("Exec loop path error %v", err)
		return err
	}
	log.Infof("Find executable path %s", path)
	// exec the command in the new namespace
	// path is the absolute path of the executable file
	// syscall.Exec won't look path in $PATH env
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		log.Errorf("syscall.Exec error %v", err)
	}
	return nil
}

// the fd 3 is the read end of the pipe
// 0 stdin
// 1 stdout
// 2 stderr
// 3 pipe
func readUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := io.ReadAll(pipe)
	if err != nil {
		log.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

func pivotRoot(root string) error {
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount rootfs to itself error: %v ", err)
	}

	// create dir in new rootfs to mnt old rootfs
	pivotDir := path.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return fmt.Errorf("mkdir .pivot_root error: %v", err)
	}

	// use syscall.PivotRoot to chroot and change root dir
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("syscall.PivotRoot error: %v", err)
	}

	// change pwd into /
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("os.Chdir error: %v", err)
	}

	pivotDir = path.Join("/", ".pivot_root")
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("syscall.Unmount .pivot_root error: %v", err)
	}

	return os.Remove(pivotDir)
}

// mount
// 1./ -> rootfs
// 2./proc
// 3./dev
func setUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Get current work dir error %v", err)
		return
	}
	log.Infof("Current work dir %s", pwd)
	if err := pivotRoot(pwd); err != nil {
		log.Errorf("Pivot root error %v", err)
		return
	}
	syscall.Mount("proc", "/proc", "proc", uintptr(syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV), "")
	syscall.Mount("tmpfs", "/dev", "tmpfs", uintptr(syscall.MS_STRICTATIME), "mode=755")
}
