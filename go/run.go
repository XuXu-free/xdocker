package main

import (
	"os"
	"strings"
	"xdocker/cgroups"
	"xdocker/cgroups/subsystems"
	"xdocker/container"

	log "github.com/sirupsen/logrus"
)

func Run(tty bool, comArray []string, resConf *subsystems.ResourceConfig) {
	parent, writePipe := container.NewParentProcess(tty)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}
	cgroupManager := cgroups.NewCgroupManager("xu-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(resConf)
	cgroupManager.Apply(parent.Process.Pid)
	sendInitCommand(comArray, writePipe)
	parent.Wait()
	mntURL := "/app/aufs/mnt/"
	rootURL := "/app/aufs/"
	container.DeleteWorkSpace(rootURL, mntURL)
	os.Exit(0)
}

// write init command to the write end of the pipe
func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command write to pip: %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}
