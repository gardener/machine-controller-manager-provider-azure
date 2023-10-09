/*
SPDX-FileCopyrightText:  2014 The Kubernetes Authors.

SPDX-License-Identifier: Apache-2.0

This file was copied and modified from the kubernetes/kubernetes project
https://github.com/kubernetes/kubernetes/release-1.8/cmd/kube-controller-manager/controller_manager.go

Modifications:
SPDX-FileCopyrightText: 2017 SAP SE or an SAP affiliate company and Gardener contributors
*/

package main

import (
	"fmt"
	"os"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/provider"
	_ "github.com/gardener/machine-controller-manager/pkg/util/client/metrics/prometheus" // for access metric registration
	"github.com/gardener/machine-controller-manager/pkg/util/provider/app"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/app/options"
	_ "github.com/gardener/machine-controller-manager/pkg/util/reflector/prometheus" // for reflector metric registration
	_ "github.com/gardener/machine-controller-manager/pkg/util/workqueue/prometheus" // for workqueue metric registration
	"github.com/spf13/pflag"
	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
)

func main() {

	s := options.NewMCServer()
	s.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	driver := provider.NewDefaultDriver(access.NewDefaultAccessFactory())
	if err := app.Run(s, driver); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
