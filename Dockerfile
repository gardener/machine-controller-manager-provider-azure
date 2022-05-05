# SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#############      builder                                  #############
FROM golang:1.13.5 AS builder

WORKDIR /go/src/github.com/gardener/machine-controller-manager-provider-azure
COPY . .

RUN .ci/build

#############      base                                     #############
FROM eu.gcr.io/gardener-project/3rd/alpine:3.15.4 as base

RUN apk add --update bash curl tzdata
WORKDIR /

#############      machine-controller               #############
FROM base AS machine-controller

COPY --from=builder /go/src/github.com/gardener/machine-controller-manager-provider-azure/bin/rel/machine-controller /machine-controller
ENTRYPOINT ["/machine-controller"]
