// Copyright © 2019 Percona, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/Percona-Lab/percona-dbaas-cli/dbaas"
	"github.com/Percona-Lab/percona-dbaas-cli/dbaas/pxc"
)

// bcpCmd represents the list command
var bcpCmd = &cobra.Command{
	Use:   "create-backup <pxc-cluster-name>",
	Short: "Create MySQL backup",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("You have to specify pxc-cluster-name")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		sp := spinner.New(spinner.CharSets[14], 250*time.Millisecond)
		sp.Color("green", "bold")
		sp.Prefix = "Looking for the cluster..."
		sp.FinalMSG = ""
		sp.Start()
		defer sp.Stop()

		ext, err := dbaas.IsObjExists("pxc", name)

		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] check if cluster exists: %v\n", err)
			return
		}

		if !ext {
			sp.Stop()
			fmt.Fprintf(os.Stderr, "Unable to find cluster \"%s/%s\"\n", "pxc", name)
			list, err := dbaas.List("pxc")
			if err != nil {
				return
			}
			fmt.Println("Avaliable clusters:")
			fmt.Print(list)
			return
		}

		sp.Prefix = "Creating backup..."

		bcp := pxc.NewBackup(name)

		bcp.Setup("fs-pvc")

		ok := make(chan string)
		msg := make(chan dbaas.OutuputMsg)
		cerr := make(chan error)

		go dbaas.Backup("pxc-backup", bcp, ok, msg, cerr)
		tckr := time.NewTicker(1 * time.Second)
		defer tckr.Stop()
		for {
			select {
			case okmsg := <-ok:
				sp.FinalMSG = fmt.Sprintf("Creating backup...[done]\n%s\n", okmsg)
				return
			case omsg := <-msg:
				switch omsg.(type) {
				case dbaas.OutuputMsgDebug:
					// fmt.Printf("\n[debug] %s\n", omsg)
				case dbaas.OutuputMsgError:
					sp.Stop()
					fmt.Printf("[operator log error] %s\n", omsg)

					sp.Start()
				}
			case err := <-cerr:
				fmt.Fprintf(os.Stderr, "\n[ERROR] create backup: %v\n", err)
				return
			}
		}
	},
}

func init() {
	pxcCmd.AddCommand(bcpCmd)
}
