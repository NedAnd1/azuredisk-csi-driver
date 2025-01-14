/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// fileCmd represents the file command
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "A pod's logs file",
	Long:  `A pod's logs file`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("file name is a required argument for \"file\" command")
			os.Exit(0)
		}
		filePath := args[0]
		volumes, nodes, requestIds, since, sinceTime, isFollow, isPrevious := GetFlags(cmd)

		if isFollow {
			fmt.Println("file source can't work with --follow")
			os.Exit(0)
		}
		if isPrevious {
			fmt.Println("file source can't work with --previous")
			os.Exit(0)
		}

		GetLogsByFile(filePath, volumes, nodes, requestIds, since, sinceTime)
	},
}

func init() {
	getCmd.AddCommand(fileCmd)
}

func GetLogsByFile(path string, volumes []string, nodes []string, requestIds []string, since string, sinceTime string) {
	if sinceTime != "" {
		t, err := TimestampFormatValidation(sinceTime)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		t = t.UTC()
		sinceTime = t.Format("0102 15:04:05.000000")
	} else if since != "" {
		d, err := time.ParseDuration(since)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		currTime := time.Now().UTC()
		sinceTime = currTime.Add(-d).Format("0102 15:04:05.000000")
	}

	// Open file
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Read file
	buf := bufio.NewScanner(file)

	LogFilter(buf, volumes, nodes, requestIds, sinceTime)
}
