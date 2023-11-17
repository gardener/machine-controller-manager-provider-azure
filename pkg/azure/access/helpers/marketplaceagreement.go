// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	mktPlaceAgreementGetServiceLabel    = "market_place_agreement_get"
	mktPlaceAgreementCreateServiceLabel = "market_place_agreement_create"
)

// GetAgreementTerms fetches the agreement terms for the purchase plan.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func GetAgreementTerms(ctx context.Context, mktPlaceAgreementAccess *armmarketplaceordering.MarketplaceAgreementsClient, purchasePlan armcompute.PurchasePlan) (agreementTerms *armmarketplaceordering.AgreementTerms, err error) {
	defer instrument.RecordAzAPIMetric(err, mktPlaceAgreementGetServiceLabel, time.Now())
	resp, err := mktPlaceAgreementAccess.Get(ctx, armmarketplaceordering.OfferTypeVirtualmachine, *purchasePlan.Publisher, *purchasePlan.Product, *purchasePlan.Name, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to get marketplace agreement for PurchasePlan: %+v", purchasePlan)
		return nil, err
	}
	agreementTerms = &resp.AgreementTerms
	return
}

// AcceptAgreement updates the agreementTerms as accepted.
// NOTE: All calls to this Azure API are instrumented as prometheus metric.
func AcceptAgreement(ctx context.Context, mktPlaceAgreementAccess *armmarketplaceordering.MarketplaceAgreementsClient, purchasePlan armcompute.PurchasePlan, existingAgreement armmarketplaceordering.AgreementTerms) (err error) {
	defer instrument.RecordAzAPIMetric(err, mktPlaceAgreementCreateServiceLabel, time.Now())
	updatedAgreement := existingAgreement
	updatedAgreement.Properties.Accepted = to.Ptr(true)
	_, err = mktPlaceAgreementAccess.Create(ctx, armmarketplaceordering.OfferTypeVirtualmachine, *purchasePlan.Publisher, *purchasePlan.Product, *purchasePlan.Name, updatedAgreement, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to create marketplace agreement for PurchasePlan: %+v", purchasePlan)
	}
	return
}
