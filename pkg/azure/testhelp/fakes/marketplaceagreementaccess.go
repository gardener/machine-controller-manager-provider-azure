package fakes

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering"
	fakemktplaceordering "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/marketplaceordering/armmarketplaceordering/fake"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
)

type MarketPlaceAgreementAccessBuilder struct {
	server          fakemktplaceordering.MarketplaceAgreementsServer
	clusterState    *ClusterState
	apiBehaviorSpec *APIBehaviorSpec
}

func (b *MarketPlaceAgreementAccessBuilder) WithClusterState(clusterState *ClusterState) *MarketPlaceAgreementAccessBuilder {
	b.clusterState = clusterState
	return b
}

func (b *MarketPlaceAgreementAccessBuilder) WithAPIBehaviorSpec(apiBehaviorSpec *APIBehaviorSpec) *MarketPlaceAgreementAccessBuilder {
	b.apiBehaviorSpec = apiBehaviorSpec
	return b
}

func (b *MarketPlaceAgreementAccessBuilder) withGet() *MarketPlaceAgreementAccessBuilder {
	b.server.Get = func(ctx context.Context, offerType armmarketplaceordering.OfferType, publisherID string, offerID string, planID string, options *armmarketplaceordering.MarketplaceAgreementsClientGetOptions) (resp azfake.Responder[armmarketplaceordering.MarketplaceAgreementsClientGetResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.providerSpec.ResourceGroup, to.Ptr(MarketPlaceOrderingOfferType), testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		// planID is today kind of useless. It is never used to get an existing agreement. See https://github.com/Azure/azure-sdk-for-go/issues/21286
		agreementTerms := b.clusterState.GetAgreementTerms(offerType, publisherID, offerID)
		if agreementTerms == nil {
			// Instead of returning 404 the client returns 400 bad request. See https://github.com/Azure/azure-sdk-for-go/issues/21286
			errResp.SetError(testhelp.BadRequestError(testhelp.ErrorBadRequest))
		}
		resp.SetResponse(http.StatusOK, armmarketplaceordering.MarketplaceAgreementsClientGetResponse{AgreementTerms: *agreementTerms}, nil)
		return
	}
	return b
}

// withCreate creates a fake implementation of Create method on the underline fake server.
// In MCM-Provider-Azure this method is used to only update and not to create. The fake is tailored towards Update use cases only - It assumes that an agreement is already existing.
func (b *MarketPlaceAgreementAccessBuilder) withCreate() *MarketPlaceAgreementAccessBuilder {
	b.server.Create = func(ctx context.Context, offerType armmarketplaceordering.OfferType, publisherID string, offerID string, planID string, parameters armmarketplaceordering.AgreementTerms, options *armmarketplaceordering.MarketplaceAgreementsClientCreateOptions) (resp azfake.Responder[armmarketplaceordering.MarketplaceAgreementsClientCreateResponse], errResp azfake.ErrorResponder) {
		if b.apiBehaviorSpec != nil {
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.providerSpec.ResourceGroup, to.Ptr(MarketPlaceOrderingOfferType), testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		agreementTerms := b.clusterState.GetAgreementTerms(offerType, publisherID, offerID)
		// we expect that the ClusterState should already be configured with AgreementTerms. In MCM-Provider-Azure we assume that the customer will create an alert.
		if agreementTerms == nil {
			errResp.SetError(testhelp.ResourceNotFoundErr(testhelp.ErrorCodeResourceNotFound))
			return
		}
		b.clusterState.agreementTerms = &parameters
		resp.SetResponse(http.StatusOK, armmarketplaceordering.MarketplaceAgreementsClientCreateResponse{AgreementTerms: parameters}, nil)
		return
	}
	return b
}

func (b *MarketPlaceAgreementAccessBuilder) Build() (*armmarketplaceordering.MarketplaceAgreementsClient, error) {
	return armmarketplaceordering.NewMarketplaceAgreementsClient(testhelp.SubscriptionID, azfake.NewTokenCredential(), &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fakemktplaceordering.NewMarketplaceAgreementsServerTransport(&b.server),
		},
	})
}
