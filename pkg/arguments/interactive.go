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

// This file contains functions to prompt for flags interactively.

package arguments

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const questionAnnotationKey = "flag_survey_question"

// AddFlag adds the interactive flag to the given set of command line flags.
func AddInteractiveFlag(flags *pflag.FlagSet, value *bool) {
	flags.BoolVarP(
		value,
		"interactive",
		"i",
		false,
		"Enable interactive mode.",
	)
}

// SetQuestion sets a friendlier text to use when prompting instead of flag name.
func SetQuestion(fs *pflag.FlagSet, flagName, question string) {
	fs.SetAnnotation(flagName, questionAnnotationKey, []string{question})
}

// GetQuestion returns the text set by SetQuestion, or fallback based on flag name.
func getQuestion(flag *pflag.Flag) string {
	values, ok := flag.Annotations[questionAnnotationKey]
	if ok && len(values) >= 1 {
		return values[0]
	}
	// Capitalize first word
	words := strings.Split(flag.Name, "-")
	words[0] = strings.Title(words[0])
	return strings.Join(words, " ") + ":"
}

// OptionsFunc is a signature for functions generating arrays of values.
// They will be used both for interactive mode and flag completions.
type OptionsFunc func(connection *sdk.Connection) ([]Option, error)

// Option represents a value that can be used in interactive Select menu,
// or shell completion.
type Option struct {
	Value string
	// Optional extra text to show (not always supported).
	Description string
}

// CobraCompletionFunc is the signature cobra.RegisterFlagCompletionFunc() wants.
type CobraCompletionFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

func MakeCompleteFunc(optionsFunc OptionsFunc) CobraCompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		connection, err := ocm.NewConnection().Build()
		if err != nil {
			cobra.CompErrorln(fmt.Sprintf("unable to create API connection: %s", err))
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}
		defer connection.Close()

		options, err := optionsFunc(connection)
		if err != nil {
			cobra.CompErrorln(err.Error())
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}

		completions := []string{}
		for _, option := range options {
			// Cobra uses \t char to separate values from optional descriptions.
			valueTabDescription := option.Value
			if option.Description != "" {
				valueTabDescription += "\t" + option.Description
			}
			completions = append(completions, valueTabDescription)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

// TODO: support custom flag types.
//   Some functions below could work with any flag type, if we remove GetString() etc. enforcment.

// TODO: support required flags?
//   It's too late to call interactive functions from Run() because Cobra will exit if not provided
//   on command line.  But probably could work from PreRun()?

// ifInteractive is a helper running the given function if --interactive is set.
func ifInteractive(fs *pflag.FlagSet, then func() error) error {
	interactive, err := fs.GetBool("interactive")
	if err != nil {
		return fmt.Errorf(`no such flag "interactive"`)
	}
	if !interactive {
		return nil
	}
	return then()
}

// PromptBool sets a bool flag value interactively, unless already set.
// Does nothing in non-interactive mode.
func PromptBool(fs *pflag.FlagSet, flagName string) error {
	value, err := fs.GetBool(flagName)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	flag := fs.Lookup(flagName)

	return ifInteractive(fs, func() error {
		if !flag.Changed {
			prompt := &survey.Confirm{
				Message: getQuestion(flag),
				Help:    flag.Usage,
				Default: value,
			}
			var response bool
			err = survey.AskOne(prompt, &response)
			if err != nil {
				return err
			}
			flag.Value.Set(strconv.FormatBool(response))
		}
		return nil
	})
}

// PromptInt sets an integer flag value interactively, unless already set.
// validation func is optional, and runs after the flag is already set.
// Does nothing in non-interactive mode. TODO: always validate, prompt if bad.
func PromptInt(fs *pflag.FlagSet, flagName string, validate func() error) error {
	_, err := fs.GetInt(flagName)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	flag := fs.Lookup(flagName)

	return ifInteractive(fs, func() error {
		if !flag.Changed {
			var response int
			prompt := &survey.Input{
				Message: getQuestion(flag),
				Help:    flag.Usage,
				Default: flag.Value.String(),
			}
			// Set() flag as side effect of validation => prompts again if invalid.
			validator := func(val interface{}) error {
				str := val.(string)
				err := flag.Value.Set(str)
				if err != nil {
					return err
				}
				if validate != nil {
					return validate()
				}
				return nil
			}
			return survey.AskOne(prompt, &response, survey.WithValidator(validator))
		}
		return nil
	})
}

// PromptString sets a string flag value interactively, unless already set.
// Does nothing in non-interactive mode.
func PromptString(fs *pflag.FlagSet, flagName string) error {
	value, err := fs.GetString(flagName)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	flag := fs.Lookup(flagName)

	return ifInteractive(fs, func() error {
		if !flag.Changed {
			var response string
			prompt := &survey.Input{
				Message: getQuestion(flag),
				Help:    flag.Usage,
				Default: value,
			}
			err = survey.AskOne(prompt, &response)
			if err != nil {
				return err
			}
			flag.Value.Set(response)
		}
		return nil
	})
}

// PromptPassword sets a sensitive string flag value interactively, unless already set.
// Does nothing in non-interactive mode.
func PromptPassword(fs *pflag.FlagSet, flagName string) error {
	// Don't need value but make sure it's a string flag.
	_, err := fs.GetString(flagName)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	flag := fs.Lookup(flagName)

	return ifInteractive(fs, func() error {
		if !flag.Changed {
			var response string
			prompt := &survey.Password{
				Message: getQuestion(flag),
				Help:    flag.Usage,
			}
			err = survey.AskOne(prompt, &response)
			if err != nil {
				return err
			}
			flag.Value.Set(response)
		}
		return nil
	})
}

// PromptFilePath sets a FilePath flag value interactively, unless already set.
// Does nothing in non-interactive mode.
func PromptFilePath(fs *pflag.FlagSet, flagName string) error {
	flag := fs.Lookup(flagName)
	if flag.Value.Type() != "filepath" {
		return fmt.Errorf("PromptFilePath can't be used on flag %q of type %q",
			flagName, flag.Value.Type())
	}

	return ifInteractive(fs, func() error {
		if !flag.Changed {
			prompt := &survey.Input{
				Message: getQuestion(flag),
				Help:    flag.Usage,
				Default: flag.Value.String(),
				Suggest: func(toComplete string) []string {
					files, _ := filepath.Glob(toComplete + "*")
					return files
				},
			}
			var response string
			err := survey.AskOne(prompt, &response)
			if err != nil {
				return err
			}
			flag.Value.Set(response)
		}
		return nil
	})
}

// PromptIPNet sets an optional IPNet flag value interactively, unless already set.
// Does nothing in non-interactive mode.
func PromptIPNet(fs *pflag.FlagSet, flagName string) error {
	flag := fs.Lookup(flagName)
	if flag.Value.Type() != "ipNet" {
		return fmt.Errorf("PromptIPNet can't be used on flag %q of type %q",
			flagName, flag.Value.Type())
	}

	return ifInteractive(fs, func() error {
		if flag.Changed {
			return nil
		}

		prompt := &survey.Input{
			Message: getQuestion(flag),
			Help:    flag.Usage,
			// TODO respect flag default, if set
			// (awkward because https://github.com/golang/go/issues/39516).
		}
		var response string
		// Set() flag as side effect of validation => prompts again if invalid.
		validator := func(val interface{}) error {
			str := val.(string)
			// Accept empty string to allow keeping the IPNet unset.
			if str == "" {
				return nil
			}
			return flag.Value.Set(str)
		}
		return survey.AskOne(prompt, &response, survey.WithValidator(validator))
	})
}

// PromptOrCheckOneOf uses a set of valid options for interactive prompt and/or validation.
// When flag is already set, validates it and doesn't prompt. TODO: prompt if bad.
// When not interactive, allows an optional flag to remain omitted.
func PromptOrCheckOneOf(fs *pflag.FlagSet, flagName string, options []Option) error {
	err := PromptOneOf(fs, flagName, options)
	if err != nil {
		return err
	}
	return CheckOneOf(fs, flagName, options)
}

// PromptOneOf sets a string flag value interactively, from a set of valid options,
// unless already set.
func PromptOneOf(fs *pflag.FlagSet, flagName string, options []Option) error {
	return ifInteractive(fs, func() error {
		return doPromptOneOf(fs, flagName, options)
	})
}

func doPromptOneOf(fs *pflag.FlagSet, flagName string, options []Option) error {
	value, err := fs.GetString(flagName)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	flag := fs.Lookup(flagName)

	// A flag may have a default in non-interactive mode, but still be worth prompting
	// in interactive mode unless explictily specified on command line.
	if !flag.Changed {
		optionValues := make([]string, len(options))
		for i, option := range options {
			optionValues[i] = option.Value
		}

		prompt := &survey.Select{
			Message: getQuestion(flag),
			Help:    flag.Usage,
			Options: optionValues,
			Default: value,
		}
		var response string
		err = survey.AskOne(prompt, &response)
		if err != nil {
			return err
		}
		flag.Value.Set(response)
	}
	return nil
}

// CheckOneOf returns error if flag has been set and is not one of given options.
// It's appropriate for both optional flags (no error not given)
// and required flags (Cobra validated they're given before command .Run).
func CheckOneOf(fs *pflag.FlagSet, flagName string, options []Option) error {
	if fs.Changed(flagName) {
		return requireOneOf(fs, flagName, options)
	}
	return nil
}

// requireOneOf returns error if flag is not one of given options.
func requireOneOf(fs *pflag.FlagSet, flagName string, options []Option) error {
	flag := fs.Lookup(flagName)
	if flag == nil {
		return fmt.Errorf("no such flag %q", flagName)
	}

	value := flag.Value.String()
	for _, option := range options {
		if value == option.Value {
			return nil
		}
	}
	return fmt.Errorf("A valid --%s must be specified.\nValid options: %+v", flagName, options)
}
