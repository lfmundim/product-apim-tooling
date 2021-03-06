/*
*  Copyright (c) WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
*
*  WSO2 Inc. licenses this file to you under the Apache License,
*  Version 2.0 (the "License"); you may not use this file except
*  in compliance with the License.
*  You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing,
* software distributed under the License is distributed on an
* "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
* KIND, either express or implied.  See the License for the
* specific language governing permissions and limitations
* under the License.
 */

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
)

const uninstallCmdLiteral = "uninstall"
const uninstallCmdShortDesc = "Uninstall an operator"
const uninstallCmdLongDesc = "Uninstall an operator in the configured K8s cluster"
const uninstallCmdExamples = utils.ProjectName + ` ` + uninstallCmdLiteral + ` ` + uninstallApiOperatorCmdLiteral

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:     uninstallCmdLiteral,
	Short:   uninstallCmdShortDesc,
	Long:    uninstallCmdLongDesc,
	Example: uninstallCmdExamples,
}

func init() {
	RootCmd.AddCommand(uninstallCmd)
}
