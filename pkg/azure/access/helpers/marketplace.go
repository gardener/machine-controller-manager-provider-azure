package helpers

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/access/errors"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/instrument"
)

const (
	mktPlaceAgreementGetServiceLabel    = "market_place_agreement_get"
	mktPlaceAgreementCreateServiceLabel = "market_place_agreement_create"
)

func GetAgreementTerms(ctx context.Context, mktPlaceAgreementAccess *armmarketplaceordering.MarketplaceAgreementsClient, purchasePlan armcompute.PurchasePlan) (agreementTerms *armmarketplaceordering.AgreementTerms, err error) {
	defer instrument.RecordAzAPIMetric(err, mktPlaceAgreementGetServiceLabel, time.Now())
	resp, err := mktPlaceAgreementAccess.GetAgreement(ctx, *purchasePlan.Publisher, *purchasePlan.Product, *purchasePlan.Name, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to get marketplace agreement for PurchasePlan: %+v", purchasePlan)
		return nil, err
	}
	agreementTerms = &resp.AgreementTerms
	return
}

func CreateAgreement(ctx context.Context, mktPlaceAgreementAccess *armmarketplaceordering.MarketplaceAgreementsClient, purchasePlan armcompute.PurchasePlan, agreementTerms armmarketplaceordering.AgreementTerms) (err error) {
	defer instrument.RecordAzAPIMetric(err, mktPlaceAgreementCreateServiceLabel, time.Now())
	_, err = mktPlaceAgreementAccess.Create(ctx, armmarketplaceordering.OfferTypeVirtualmachine, *purchasePlan.Publisher, *purchasePlan.Product, *purchasePlan.Name, agreementTerms, nil)
	if err != nil {
		errors.LogAzAPIError(err, "Failed to create marketplace agreement for PurchasePlan: %+v", purchasePlan)
	}
	return
}
