package fakes

import (
	"context"

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
			err := b.apiBehaviorSpec.SimulateForResourceType(ctx, b.clusterState.ResourceGroup, to.Ptr(MarketPlaceOrderingOfferType), testhelp.AccessMethodGet)
			if err != nil {
				errResp.SetError(err)
				return
			}
		}
		agreementTerms := b.clusterState.GetAgreementTerms(offerType, publisherID, offerID, planID)
		if agreementTerms == nil {
			//errResp.SetError(testhelp.ResourceNotFoundErr())
		}
		return
	}
	return b
}

func (b *MarketPlaceAgreementAccessBuilder) withCreate() *MarketPlaceAgreementAccessBuilder {
	b.server.Create = func(ctx context.Context, offerType armmarketplaceordering.OfferType, publisherID string, offerID string, planID string, parameters armmarketplaceordering.AgreementTerms, options *armmarketplaceordering.MarketplaceAgreementsClientCreateOptions) (resp azfake.Responder[armmarketplaceordering.MarketplaceAgreementsClientCreateResponse], errResp azfake.ErrorResponder) {
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
