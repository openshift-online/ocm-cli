/*
Copyright (c) 2019 Red Hat, Inc.

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

package completion

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion scripts for various shells",
	Long: `To load completions:

Bash:

$ source <(ocm completion bash)

# To load completions for each session, execute once:
Linux:
  $ ocm completion bash > /etc/bash_completion.d/ocm
MacOS:
  $ ocm completion bash > /usr/local/etc/bash_completion.d/ocm

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ ocm completion zsh > "${fpath[1]}/_ocm"

# You will need to start a new shell for this setup to take effect.

Fish:

$ ocm completion fish | source

# To load completions for each session, execute once:
$ ocm completion fish > ~/.config/fish/completions/ocm.fish

P.S. Debugging completion logic:
- Set BASH_COMP_DEBUG_FILE env var to enable logging to that file.
- See https://github.com/spf13/cobra/blob/master/shell_completions.md.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// backward compatibility (previously only supported bash, took no args)
		if len(args) == 0 {
			args = []string{"bash"}
		}

		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		default:
			return fmt.Errorf("invalid shell %q", args[0])
		}
		return nil
	},
}
