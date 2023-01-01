package allocator

import (
	_ "embed"
	"github.com/keyval-dev/odigos/odiglet/pkg/allocator/bj"
	"github.com/keyval-dev/odigos/odiglet/pkg/allocator/debug/elf"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"os"
)

//go:embed payload/mmap
var payloadBytes []byte

func Apply(exePath string) error {
	log.Logger.V(0).Info("Applying allocation", "exePath", exePath)

	inputStat, err := os.Stat(exePath)
	if err != nil {
		log.Logger.Error(err, "Failed to stat exePath", "exePath", exePath)
		return err
	}

	// Parse elf file
	f, err := elf.Open(exePath)
	if err != nil {
		log.Logger.Error(err, "Failed to open elf file", "exePath", exePath)
		return err
	}

	output, shoff, err := bj.StaticSilvioMethod(f, payloadBytes)
	if err != nil {
		log.Logger.Error(err, "Failed to apply allocation", "exePath", exePath)
		return err
	}

	output = applyFixForGoBinaries(f, shoff, output)

	// Write output to exePath
	err = os.WriteFile(exePath, output, inputStat.Mode())
	if err != nil {
		log.Logger.Error(err, "Failed to write output to exePath", "exePath", exePath)
		return err
	}

	return nil
}
