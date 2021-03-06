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
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/wso2/product-apim-tooling/import-export-cli/credentials"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
)

var importAppFile string
var importAppEnvironment string
var importAppOwner string
var preserveOwner bool
var skipSubscriptions bool
var importAppSkipKeys bool
var importAppUpdateApplication bool

// ImportApp command related usage info
const importAppCmdLiteral = "import-app"
const importAppCmdShortDesc = "Import App"

const importAppCmdLongDesc = "Import an Application to an environment"

const importAppCmdExamples = utils.ProjectName + ` ` + importAppCmdLiteral + ` -f qa/apps/sampleApp.zip -e dev
` + utils.ProjectName + ` ` + importAppCmdLiteral + ` -f staging/apps/sampleApp.zip -e prod -o testUser
` + utils.ProjectName + ` ` + importAppCmdLiteral + ` -f qa/apps/sampleApp.zip --preserveOwner --skipSubscriptions -e prod
NOTE: Both the flags (--file (-f) and --environment (-e)) are mandatory`

// importAppCmd represents the importApp command
var ImportAppCmd = &cobra.Command{
	Use: importAppCmdLiteral + " (--file <app-zip-file> --environment " +
		"<environment-to-which-the-app-should-be-imported>)",
	Short:   importAppCmdShortDesc,
	Long:    importAppCmdLongDesc,
	Example: importAppCmdExamples,
	Run: func(cmd *cobra.Command, args []string) {
		utils.Logln(utils.LogPrefixInfo + importAppCmdLiteral + " called")
		var appsExportDirectory = filepath.Join(utils.ExportDirectory, utils.ExportedAppsDirName)
		cred, err := getCredentials(importAppEnvironment)
		if err != nil {
			utils.HandleErrorAndExit("Error getting credentials", err)
		}
		executeImportAppCmd(cred, importAppOwner, appsExportDirectory)
	},
}

func executeImportAppCmd(credential credentials.Credential, importAppOwner, exportDirectory string) {
	accessToken, err := credentials.GetOAuthAccessToken(credential, importAppEnvironment)
	if err != nil {
		utils.HandleErrorAndExit("Error getting OAuth Tokens", err)
	}

	adminEndpoint := utils.GetAdminEndpointOfEnv(importAppEnvironment, utils.MainConfigFilePath)
	resp, err := ImportApplication(importAppFile, importAppOwner, adminEndpoint, accessToken, exportDirectory)
	if err != nil {
		utils.HandleErrorAndExit("Error importing Application", err)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		// 200 OK or 201 Created
		utils.Logln(utils.LogPrefixInfo+"Header:", resp.Header)
		fmt.Println("Successfully imported Application!")
	} else if resp.StatusCode == http.StatusMultiStatus {
		// 207 Multi Status
		fmt.Printf("\nPartially imported Application" +
			"\nNOTE: One or more subscriptions were not imported due to unavailability of APIs/Tiers\n")
	} else if resp.StatusCode == http.StatusUnauthorized {
		// 401 Unauthorized
		fmt.Println("Invalid Credentials or You may not have enough permission!")
	} else if resp.StatusCode == http.StatusForbidden {
		// 401 Unauthorized
		fmt.Printf("Invalid Owner!" + "\nNOTE: Cross Tenant Imports are not allowed!\n")
	} else {
		fmt.Println("Error importing Application")
		utils.Logln(utils.LogPrefixError + resp.Status)
	}
}

// ImportApplication function is used with import-app command
// @param name: name of the Application (zipped file) to be imported
// @param apiManagerEndpoint: API Manager endpoint for the environment
// @param accessToken: OAuth2.0 access token for the resource being accessed
func ImportApplication(filename, appOwner, adminEndpiont, accessToken, exportDirectory string) (*http.Response, error) {
	adminEndpiont = utils.AppendSlashToString(adminEndpiont)

	applicationImportEndpoint := adminEndpiont + "import/applications"
	url := applicationImportEndpoint + "?appOwner=" + appOwner + utils.SearchAndTag + "preserveOwner=" +
		strconv.FormatBool(preserveOwner) + utils.SearchAndTag + "skipSubscriptions=" +
		strconv.FormatBool(skipSubscriptions) + utils.SearchAndTag + "skipApplicationKeys=" + strconv.FormatBool(importAppSkipKeys) +
		utils.SearchAndTag + "update=" + strconv.FormatBool(importAppUpdateApplication)
	utils.Logln(utils.LogPrefixInfo + "Import URL: " + applicationImportEndpoint)

	zipFilePath, err := resolveImportFilePath(filename, exportDirectory)
	if err != nil {
		utils.HandleErrorAndExit("Error creating request.", err)
	}
	fmt.Println("ZipFilePath:", zipFilePath)

	extraParams := map[string]string{}

	req, err := NewAppFileUploadRequest(url, extraParams, "file", zipFilePath, accessToken)
	if err != nil {
		utils.HandleErrorAndExit("Error creating request.", err)
	}

	var tr *http.Transport
	if utils.Insecure {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: utils.GetTlsConfigWithCertificate(),
		}
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(utils.HttpRequestTimeout) * time.Second,
	}

	resp, err := client.Do(req)

	if err != nil {
		utils.Logln(utils.LogPrefixError, err)
	} else {
		//var bodyContent []byte

		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK ||
			resp.StatusCode == http.StatusMultiStatus {
			// 207 Multi Status or 201 Created or 200 OK
			fmt.Printf("\nCompleted importing the Application '" + filename + "'\n")
		} else {
			fmt.Printf("\nUnable to import the Application\n")
			fmt.Println("Status: " + resp.Status)
		}

		//fmt.Println(resp.Header)
		//resp.Body.Read(bodyContent)
		//resp.Body.Close()
		//fmt.Println(bodyContent)
	}

	return resp, err
}

// NewFileUploadRequest form an HTTP Put request
// Helper function for forming multi-part form data
// Returns the formed http request and errors
func NewAppFileUploadRequest(uri string, params map[string]string, paramName, path,
	accessToken string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, uri, body)
	request.Header.Add(utils.HeaderAuthorization, utils.HeaderValueAuthBearerPrefix+" "+accessToken)
	request.Header.Add(utils.HeaderContentType, writer.FormDataContentType())
	request.Header.Add(utils.HeaderAccept, "*/*")
	request.Header.Add(utils.HeaderConnection, utils.HeaderValueKeepAlive)

	return request, err
}

func init() {
	RootCmd.AddCommand(ImportAppCmd)
	ImportAppCmd.Flags().StringVarP(&importAppFile, "file", "f", "",
		"Name of the ZIP file of the Application to be imported")
	ImportAppCmd.Flags().StringVarP(&importAppOwner, "owner", "o", "",
		"Name of the target owner of the Application as desired by the Importer")
	ImportAppCmd.Flags().StringVarP(&importAppEnvironment, "environment", "e",
		"", "Environment from the which the Application should be imported")
	ImportAppCmd.Flags().BoolVarP(&preserveOwner, "preserveOwner", "", false,
		"Preserves app owner")
	ImportAppCmd.Flags().BoolVarP(&skipSubscriptions, "skipSubscriptions", "s", false,
		"Skip subscriptions of the Application")
	ImportAppCmd.Flags().BoolVarP(&importAppSkipKeys, "skipKeys", "", false,
		"Skip importing keys of the Application")
	ImportAppCmd.Flags().BoolVarP(&importAppUpdateApplication, "update", "", false,
		"Update the Application if it is already imported")
	_ = ImportAppCmd.MarkFlagRequired("file")
	_ = ImportAppCmd.MarkFlagRequired("environment")
}
