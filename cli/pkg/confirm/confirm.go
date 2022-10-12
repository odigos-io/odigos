package confirm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Ask(text string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [Y/n]: ", text)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))

	if response == "y" || response == "yes" {
		return true, nil
	} else if response == "n" || response == "no" {
		return false, nil
	}

	return false, fmt.Errorf("%s invalid response. Type [y/n/yes/no]", response)
}
