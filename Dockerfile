# SPDX-FileCopyrightText: Contributors to the Gardener project
#
# SPDX-License-Identifier: Apache-2.0

#############      builder                          #############
FROM --platform=$BUILDPLATFORM golang:1.26.2 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src/github.com/gardener/machine-controller-manager-provider-azure
COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH .ci/build

#############      machine-controller               #############
FROM gcr.io/distroless/static-debian12:nonroot AS machine-controller
WORKDIR /

COPY --from=builder /go/src/github.com/gardener/machine-controller-manager-provider-azure/bin/rel/machine-controller /machine-controller
ENTRYPOINT ["/machine-controller"]
