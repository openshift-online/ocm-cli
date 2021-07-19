/*
Copyright (c) 2020 Red Hat, Inc.
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

package idp

import (
	"errors"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/AlecAivazis/survey/v2"
	pwdgen "github.com/m1/go-generate-password/generator"
)

func buildHtpasswdIdp(cluster *cmv1.Cluster, idpName string) (cmv1.IdentityProviderBuilder, string, error) {
	idpBuilder := cmv1.IdentityProviderBuilder{}
	message := fmt.Sprintf("Securely store your username and password.\n" +
		"If you lose these credentials, you will have to delete and recreate the IDP.\n")

	username := args.htpasswdUsername
	password := args.htpasswdPassword

	if username == "" {
		prompt := &survey.Input{
			Message: "Enter username:",
		}
		err := survey.AskOne(prompt, &username)
		if err != nil {
			return idpBuilder, "", errors.New("Expected a username")
		}
	}

	if password == "" {
		prompt := &survey.Password{
			Message: "Enter password or leave empty to generate:",
		}
		err := survey.AskOne(prompt, &password)
		if err != nil {
			return idpBuilder, "", errors.New("Expected a password")
		}
	}
	if password == "" {
		generator, err := pwdgen.NewWithDefault()
		if err != nil {
			return idpBuilder, "", errors.New("Failed to initialize password generator")
		}
		generatedPwd, err := generator.Generate()
		if err != nil {
			return idpBuilder, "", errors.New("Failed to generate a password")
		}
		password = *generatedPwd
		message += "You can now log in with the provided username and the password '" + password + "'.\n"
	}

	// Create HTPasswd IDP
	htpasswdIDP := cmv1.NewHTPasswdIdentityProvider().
		Username(username).
		Password(password)

	// Create new IDP with HTPasswd provider
	idpBuilder.
		Type("HTPasswdIdentityProvider"). // FIXME: ocm-api-model has the wrong enum values
		Name(idpName).
		MappingMethod(cmv1.IdentityProviderMappingMethod(args.mappingMethod)).
		Htpasswd(htpasswdIDP)

	return idpBuilder, message, nil
}
