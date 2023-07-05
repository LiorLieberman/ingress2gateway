/*
Copyright © 2022 Kubernetes Authors

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
package i2gw

import (
	"context"
	"fmt"
	"os"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/cli-runtime/pkg/printers"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func Run(printer printers.ResourcePrinter) {
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("failed to create client")
		os.Exit(1)
	}

	ingressList := &networkingv1.IngressList{}

	err = cl.List(context.Background(), ingressList)
	if err != nil {
		fmt.Printf("failed to list ingresses: %v\n", err)
		os.Exit(1)
	}

	httpRoutes, gateways, errors := ingresses2GatewaysAndHTTPRoutes(ingressList.Items)
	if len(errors) > 0 {
		fmt.Printf("# Encountered %d errors\n", len(errors))
		for _, err = range errors {
			fmt.Printf("# %s\n", err)
		}
		return
	}

	outputResult(printer, httpRoutes, gateways)
}

func ingresses2GatewaysAndHTTPRoutes(ingresses []networkingv1.Ingress) ([]gatewayv1beta1.HTTPRoute, []gatewayv1beta1.Gateway, field.ErrorList) {
	aggregator := ingressAggregator{ruleGroups: map[ruleGroupKey]*ingressRuleGroup{}}

	var errs field.ErrorList
	for _, ingress := range ingresses {
		errs = append(errs, aggregator.addIngress(ingress)...)
	}
	if len(errs) > 0 {
		return nil, nil, errs
	}

	return aggregator.toHTTPRoutesAndGateways()
}

func outputResult(printer printers.ResourcePrinter, httpRoutes []gatewayv1beta1.HTTPRoute, gateways []gatewayv1beta1.Gateway) {
	for i := range gateways {
		err := printer.PrintObj(&gateways[i], os.Stdout)
		if err != nil {
			fmt.Printf("# Error printing %s HTTPRoute: %v\n", gateways[i].Name, err)
		}
	}

	for i := range httpRoutes {
		err := printer.PrintObj(&httpRoutes[i], os.Stdout)
		if err != nil {
			fmt.Printf("# Error printing %s HTTPRoute: %v\n", httpRoutes[i].Name, err)
		}
	}
}
