package main

import (
	"fmt"

	"github.com/madebymany/tilly/Godeps/_workspace/src/github.com/abourget/slack"
)

func main() {
	api := slack.New("YOUR_TOKEN_HERE")
	// If you set debugging, it will log all requests to the console
	// Useful when encountering issues
	// api.SetDebug(true)
	groups, err := api.GetGroups(false)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	for _, group := range groups {
		fmt.Printf("Id: %s, Name: %s\n", group.Id, group.Name)
	}
}
