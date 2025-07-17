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

package ingress

import (
	"fmt"
	"regexp"
	"strings"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{4,5}$`)

var validLbTypes = []string{string(cmv1.LoadBalancerFlavorClassic), string(cmv1.LoadBalancerFlavorNlb)}
var ValidWildcardPolicies = []string{string(cmv1.WildcardPolicyWildcardsDisallowed),
	string(cmv1.WildcardPolicyWildcardsAllowed)}
var ValidNamespaceOwnershipPolicies = []string{string(cmv1.NamespaceOwnershipPolicyStrict),
	string(cmv1.NamespaceOwnershipPolicyInterNamespaceAllowed)}
var expectedComponentRoutes = []string{
	string(cmv1.ComponentRouteTypeOauth),
	string(cmv1.ComponentRouteTypeConsole),
	string(cmv1.ComponentRouteTypeDownloads),
}
var expectedParameters = []string{
	hostnameParameter,
	tlsSecretRefParameter,
}

var args struct {
	clusterKey    string
	private       bool
	routeSelector string
	lbType        string

	excludedNamespaces        string
	wildcardPolicy            string
	namespaceOwnershipPolicy  string
	clusterRoutesHostname     string
	clusterRoutesTlsSecretRef string

	componentRoutes string
}

const (
	privateFlag                   = "private"
	labelMatchFlag                = "label-match"
	lbTypeFlag                    = "lb-type"
	routeSelectorFlag             = "route-selector"
	excludedNamespacesFlag        = "excluded-namespaces"
	wildcardPolicyFlag            = "wildcard-policy"
	namespaceOwnershipPolicyFlag  = "namespace-ownership-policy"
	clusterRoutesHostnameFlag     = "cluster-routes-hostname"
	clusterRoutesTlsSecretRefFlag = "cluster-routes-tls-secret-ref"
	componentRoutesFlag           = "component-routes"

	expectedLengthOfParsedComponent = 2
	hostnameParameter               = "hostname"
	//nolint:gosec
	tlsSecretRefParameter = "tlsSecretRef"
)

var Cmd = &cobra.Command{
	Use:     "ingress --cluster={NAME|ID|EXTERNAL_ID} [flags] INGRESS_ID",
	Aliases: []string{"route", "routes", "ingresses"},
	Short:   "Edit a cluster Ingress",
	Long:    "Edit an Ingress endpoint to determine access to the cluster.",
	Example: `  #  Update the router selectors for the additional ingress with ID 'a1b2'
  ocm edit ingress --label-match=foo=bar --cluster=mycluster a1b2
  #  Update the default ingress using the sub-domain identifier
  ocm edit ingress --private=false --cluster=mycluster apps"`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID or external_id of the cluster to add the ingress to (required).",
	)
	//nolint:gosec
	Cmd.MarkFlagRequired("cluster")

	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict application route to direct, private connectivity.",
	)

	flags.StringVar(
		&args.routeSelector,
		labelMatchFlag,
		"",
		fmt.Sprintf("Alias to '%s' flag.", routeSelectorFlag),
	)

	flags.StringVar(
		&args.routeSelector,
		routeSelectorFlag,
		"",
		"Route Selector for ingress. Format should be a comma-separated list of 'key=value'. "+
			"If no label is specified, all routes will be exposed on both routers."+
			" For legacy ingress support these are inclusion labels, otherwise they are treated as exclusion label.",
	)

	flags.StringVar(
		&args.lbType,
		lbTypeFlag,
		"",
		fmt.Sprintf("Type of Load Balancer. Options are %s.", strings.Join(validLbTypes, ",")),
	)

	flags.StringVar(
		&args.excludedNamespaces,
		excludedNamespacesFlag,
		"",
		"Excluded namespaces for ingress. Format should be a comma-separated list 'value1, value2...'. "+
			"If no values are specified, all namespaces will be exposed.",
	)

	flags.StringVar(
		&args.wildcardPolicy,
		wildcardPolicyFlag,
		"",
		fmt.Sprintf("Wildcard Policy for ingress. Options are %s", strings.Join(ValidWildcardPolicies, ",")),
	)

	flags.StringVar(
		&args.namespaceOwnershipPolicy,
		namespaceOwnershipPolicyFlag,
		"",
		fmt.Sprintf("Namespace Ownership Policy for ingress. Options are %s",
			strings.Join(ValidNamespaceOwnershipPolicies, ",")),
	)

	flags.StringVar(
		&args.clusterRoutesHostname,
		clusterRoutesHostnameFlag,
		"",
		"Components route hostname for oauth, console, downloads.",
	)

	flags.StringVar(
		&args.clusterRoutesTlsSecretRef,
		clusterRoutesTlsSecretRefFlag,
		"",
		"Components route TLS secret reference for oauth, console, downloads.",
	)

	flags.StringVar(
		&args.componentRoutes,
		componentRoutesFlag,
		"",
		//nolint:lll
		"Component routes settings. Available keys [oauth, console, downloads]. For each key a pair of hostname and tlsSecretRef is expected to be supplied. "+
			"Format should be a comma separate list 'oauth: hostname=example-hostname;tlsSecretRef=example-secret-ref,downloads:...",
	)
}

func run(cmd *cobra.Command, argv []string) error {

	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one command line parameter containing the id of the ingress")
	}

	ingressID := argv[0]
	if !ingressKeyRE.MatchString(ingressID) {
		return fmt.Errorf(
			"Ingress  identifier '%s' isn't valid: it must contain only letters or digits",
			ingressID,
		)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the cluster management api
	clusterCollection := connection.ClustersMgmt().V1().Clusters()

	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf("Cluster '%s' is not yet ready", clusterKey)
	}

	ingresses, err := c.GetIngresses(clusterCollection, cluster.ID())
	if err != nil {
		return fmt.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
	}

	var ingress *cmv1.Ingress
	for _, item := range ingresses {
		if ingressID == "apps" && item.Default() {
			ingress = item
		}
		if ingressID == "apps2" && !item.Default() {
			ingress = item
		}
		if item.ID() == ingressID {
			ingress = item
		}
	}
	if ingress == nil {
		return fmt.Errorf("Failed to get ingress '%s' for cluster '%s'", ingressID, clusterKey)
	}

	ingressBuilder := cmv1.NewIngress().ID(ingress.ID())
	if cmd.Flags().Changed(privateFlag) {
		if args.private {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodInternal)
		} else {
			ingressBuilder = ingressBuilder.Listening(cmv1.ListeningMethodExternal)
		}
	}

	// Add route selectors
	if cmd.Flags().Changed(labelMatchFlag) || cmd.Flags().Changed(routeSelectorFlag) {
		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Can't edit `%s` for Hosted Control Plane clusters", routeSelectorFlag)
		}
		routeSelectors, err := GetRouteSelector(args.routeSelector)
		if err != nil {
			return err
		}
		ingressBuilder = ingressBuilder.RouteSelectors(routeSelectors)
	}

	if cmd.Flags().Changed(lbTypeFlag) {
		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Can't edit `%s` for Hosted Control Plane clusters", lbTypeFlag)
		}
		if cluster.AWS().STS().RoleARN() != "" {
			return fmt.Errorf("Can't edit `%s` for STS clusters", lbTypeFlag)
		}
		ingressBuilder = ingressBuilder.LoadBalancerType(cmv1.LoadBalancerFlavor(args.lbType))
	}

	if cmd.Flags().Changed(excludedNamespacesFlag) {
		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Can't edit `%s` for Hosted Control Plane clusters", excludedNamespacesFlag)
		}
		_excludedNamespaces := GetExcludedNamespaces(args.excludedNamespaces)
		ingressBuilder = ingressBuilder.ExcludedNamespaces(_excludedNamespaces...)
	}

	if cmd.Flags().Changed(wildcardPolicyFlag) {
		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Can't edit `%s` for Hosted Control Plane clusters", wildcardPolicyFlag)
		}
		ingressBuilder = ingressBuilder.RouteWildcardPolicy(cmv1.WildcardPolicy(args.wildcardPolicy))
	}

	if cmd.Flags().Changed(namespaceOwnershipPolicyFlag) {
		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Can't edit `%s` for Hosted Control Plane clusters", namespaceOwnershipPolicyFlag)
		}
		ingressBuilder = ingressBuilder.RouteNamespaceOwnershipPolicy(
			cmv1.NamespaceOwnershipPolicy(args.namespaceOwnershipPolicy))
	}

	if cmd.Flags().Changed(clusterRoutesHostnameFlag) {
		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Can't edit `%s` for Hosted Control Plane clusters", clusterRoutesHostnameFlag)
		}
		ingressBuilder = ingressBuilder.ClusterRoutesHostname(args.clusterRoutesHostname)
	}

	if cmd.Flags().Changed(clusterRoutesTlsSecretRefFlag) {
		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Can't edit `%s` for Hosted Control Plane clusters", clusterRoutesTlsSecretRefFlag)
		}
		ingressBuilder = ingressBuilder.ClusterRoutesTlsSecretRef(args.clusterRoutesTlsSecretRef)
	}

	if cmd.Flags().Changed(componentRoutesFlag) {
		if cluster.Hypershift().Enabled() {
			return fmt.Errorf("Can't edit `%s` for Hosted Control Plane clusters", componentRoutesFlag)
		}
		componentRoutes, err := parseComponentRoutes(args.componentRoutes)
		if err != nil {
			return fmt.Errorf("An error occurred whilst parsing the supplied component routes: %s", err)
		}
		ingressBuilder = ingressBuilder.ComponentRoutes(componentRoutes)
	}

	ingress, err = ingressBuilder.Build()
	if err != nil {
		return fmt.Errorf("Failed to edit ingress for cluster '%s': %v", clusterKey, err)
	}

	_, err = ocm.SendTypedAndHandleDeprecation(clusterCollection.
		Cluster(cluster.ID()).
		Ingresses().
		Ingress(ingress.ID()).
		Update().
		Body(ingress))
	if err != nil {
		return fmt.Errorf("Failed to edit ingress for cluster '%s': %v", clusterKey, err)
	}
	return nil
}

func parseComponentRoutes(input string) (map[string]*cmv1.ComponentRouteBuilder, error) {
	result := map[string]*cmv1.ComponentRouteBuilder{}
	input = strings.TrimSpace(input)
	components := strings.Split(input, ",")
	if len(components) != len(expectedComponentRoutes) {
		return nil, fmt.Errorf(
			"the expected amount of component routes is %d, but %d have been supplied",
			len(expectedComponentRoutes),
			len(components),
		)
	}
	for _, component := range components {
		component = strings.TrimSpace(component)
		parsedComponent := strings.Split(component, ":")
		if len(parsedComponent) != expectedLengthOfParsedComponent {
			return nil, fmt.Errorf(
				"only the name of the component should be followed by ':'",
			)
		}
		componentName := strings.TrimSpace(parsedComponent[0])
		if !utils.Contains(expectedComponentRoutes, componentName) {
			return nil, fmt.Errorf(
				"'%s' is not a valid component name. Expected include %s",
				componentName,
				utils.SliceToSortedString(expectedComponentRoutes),
			)
		}
		parameters := strings.TrimSpace(parsedComponent[1])
		componentRouteBuilder := new(cmv1.ComponentRouteBuilder)
		parsedParameter := strings.Split(parameters, ";")
		if len(parsedParameter) != len(expectedParameters) {
			return nil, fmt.Errorf(
				"only %d parameters are expected for each component",
				len(expectedParameters),
			)
		}
		for _, values := range parsedParameter {
			values = strings.TrimSpace(values)
			parsedValues := strings.Split(values, "=")
			parameterName := strings.TrimSpace(parsedValues[0])
			if !utils.Contains(expectedParameters, parameterName) {
				return nil, fmt.Errorf(
					"'%s' is not a valid parameter for a component route. Expected include %s",
					parameterName,
					utils.SliceToSortedString(expectedParameters),
				)
			}
			parameterValue := strings.TrimSpace(parsedValues[1])
			// TODO: use reflection, couldn't get it to work
			if parameterName == hostnameParameter {
				componentRouteBuilder.Hostname(parameterValue)
			} else if parameterName == tlsSecretRefParameter {
				componentRouteBuilder.TlsSecretRef(parameterValue)
			}
		}
		result[componentName] = componentRouteBuilder
	}
	return result, nil
}

func GetExcludedNamespaces(excludedNamespaces string) []string {
	if excludedNamespaces == "" {
		return []string{}
	}
	sliceExcludedNamespaces := strings.Split(excludedNamespaces, ",")
	for i := range sliceExcludedNamespaces {
		sliceExcludedNamespaces[i] = strings.TrimSpace(sliceExcludedNamespaces[i])
	}
	return sliceExcludedNamespaces
}

func GetRouteSelector(labelMatches string) (map[string]string, error) {
	if labelMatches == "" {
		return map[string]string{}, nil
	}
	routeSelectors := map[string]string{}
	for _, labelMatch := range strings.Split(labelMatches, ",") {
		if !strings.Contains(labelMatch, "=") {
			return nil, fmt.Errorf("Expected key=value format for label-match")
		}
		tokens := strings.Split(labelMatch, "=")
		routeSelectors[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
	}

	return routeSelectors, nil
}
