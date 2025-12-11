package network

import (
	"fmt"
	"os"
	"strings"

	"github.com/zenithax-cc/diting/internal/model"
)

const sysfsNet string = "/sys/class/net"

func collectNetInterfaces() ([]model.NetInterface, error) {
	dirs, err := os.ReadDir(sysfsNet)
	if err != nil {
		return nil, fmt.Errorf("read directory %s failed: %w", sysfsNet, err)
	}

	netInterfaces := make([]model.NetInterface, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		dirName := dir.Name()
		if strings.HasPrefix(dirName, "lo") || strings.HasPrefix(dirName, "loop") {
			continue
		}

		netInterfaces = append(netInterfaces, collectNetInterface(dirName))
	}

	return netInterfaces, nil
}

func collectNetInterface(name string) model.NetInterface {
	netInterface := model.NetInterface{
		DeviceName: name,
	}

	return netInterface
}
