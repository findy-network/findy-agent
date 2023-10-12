package common

import (
	"github.com/findy-network/findy-agent/core"
	sov "github.com/findy-network/findy-agent/std/sov/did"
	"github.com/hyperledger/aries-framework-go/component/models/did/endpoint"
	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/lainio/err2"
	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
	"github.com/mr-tron/base58"
)

func ID(d core.DIDDoc) string {
	switch doc := d.(type) {
	case *did.Doc:
		return doc.ID
	case *sov.Doc:
		return doc.ID
	default:
		assert.NotImplemented()
		return ""
	}
}

func Value58(doc core.DIDDoc, i int) string {
	value := Value(doc, i)
	return base58.Encode(value)
}

func Value(doc core.DIDDoc, i int) []byte {
	return VM(doc, i).Value
}

func VMs(d core.DIDDoc) []did.VerificationMethod {
	switch doc := d.(type) {
	case *did.Doc:
		return doc.VerificationMethod
	case *sov.Doc:
		retval := make([]did.VerificationMethod, len(doc.PublicKey))
		for i, auth := range doc.PublicKey {
			retval[i].Type = auth.Type
			retval[i].ID = auth.ID
			retval[i].Controller = auth.Controller
			retval[i].Value = try.To1(base58.Decode(auth.PublicKeyBase58))
		}
		return retval
	default:
		assert.NotImplemented()
		return nil
	}
}

func VM(d core.DIDDoc, i int) did.VerificationMethod {
	return VMs(d)[i]
}

func Services(d core.DIDDoc) []did.Service {
	switch doc := d.(type) {
	case *did.Doc:
		return doc.Service
	case *sov.Doc:
		servCount := len(doc.Service)
		retval := make([]did.Service, servCount)
		for i, service := range doc.Service {
			retval[i].ServiceEndpoint = endpoint.NewDIDCommV1Endpoint(service.ServiceEndpoint)
			retval[i].Type = service.Type
			retval[i].ID = service.ID
			retval[i].RoutingKeys = service.RoutingKeys
			retval[i].RecipientKeys = service.RecipientKeys
		}
		return retval
	default:
		assert.NotImplemented()
		return nil
	}
}

func getEndpointType(t interface{}) string {
	if epType, ok := t.(endpoint.EndpointType); ok {
		switch epType {
		case endpoint.DIDCommV1:
			return "did-communication"
		case endpoint.DIDCommV2:
			return "DIDCommMessaging"
		}
	}
	return ""
}

func SetServices(d core.DIDDoc, s []did.Service) {
	defer err2.Catch()

	switch doc := d.(type) {
	case *did.Doc:
		doc.Service = s
	case *sov.Doc:
		servCount := len(s)
		retval := make([]sov.Service, servCount)
		for i, service := range s {
			retval[i].ServiceEndpoint = try.To1(service.ServiceEndpoint.URI())
			retval[i].Type = getEndpointType(service.Type)
			retval[i].ID = service.ID
			retval[i].RoutingKeys = service.RoutingKeys
			retval[i].RecipientKeys = service.RecipientKeys
		}
		doc.Service = retval
	default:
		assert.NotImplemented()
	}
}

func Authentications(d core.DIDDoc) []did.Verification {
	switch doc := d.(type) {
	case *did.Doc:
		return doc.Authentication
	case *sov.Doc:
		retval := make([]did.Verification, len(doc.Authentication))
		for i, auth := range doc.Authentication {
			retval[i].VerificationMethod.Type = auth.Type
			retval[i].VerificationMethod.ID = auth.PublicKey
		}
		return retval
	default:
		assert.NotImplemented()
		return nil
	}
}

func Service(d core.DIDDoc, i int) did.Service {
	return Services(d)[i]
}

func RoutingKeys(d core.DIDDoc, i int) []string {
	service := Service(d, i)
	return service.RoutingKeys
}

func RecipientKeys(d core.DIDDoc, i int) []string {
	service := Service(d, i)
	return service.RecipientKeys
}
