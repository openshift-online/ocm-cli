/*
Copyright (c) 2026 Red Hat

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

package kubeletconfig

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmWorkerNodeReboot prompts the user to confirm that they accept
// a worker node reboot before the operation proceeds. verb should be
// "Creating" or "Editing". Returns (true, nil) if the user confirms,
// (false, nil) if they decline, and (false, err) on a stdin read failure.
func ConfirmWorkerNodeReboot(verb string) (bool, error) {
	fmt.Printf(
		"%s a KubeletConfig for cluster will cause all non-Control Plane nodes to reboot. "+
			"This may cause disruption to your workloads. Do you wish to continue? (y/N): ",
		verb,
	)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("Failed to read confirmation input: %v", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}
