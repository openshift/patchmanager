package util

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

func AskForConfirmation() bool {
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		klog.Fatal(err)
	}

	switch strings.ToLower(response) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		fmt.Println("I'm sorry but I didn't get what you meant, please type (y)es or (n)o and then press enter:")
		return AskForConfirmation()
	}
}
