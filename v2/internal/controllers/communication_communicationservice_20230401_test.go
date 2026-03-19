/*
Copyright (c) Microsoft Corporation.
Licensed under the MIT license.
*/

package controllers_test

import (
	"testing"

	. "github.com/onsi/gomega"

	communication "github.com/Azure/azure-service-operator/v2/api/communication/v20230401"
	resources "github.com/Azure/azure-service-operator/v2/api/resources/v1api20200601"
	"github.com/Azure/azure-service-operator/v2/internal/testcommon"
	"github.com/Azure/azure-service-operator/v2/internal/util/to"
	"github.com/Azure/azure-service-operator/v2/pkg/genruntime"
)

func Test_Communication_CommunicationService_20230401_CRUD(t *testing.T) {
	t.Parallel()

	tc := globalTestContext.ForTest(t)

	rg := tc.CreateTestResourceGroupAndWait()

	// Create a CommunicationService
	commService := &communication.CommunicationService{
		ObjectMeta: tc.MakeObjectMeta("commservice"),
		Spec: communication.CommunicationService_Spec{
			Location:     to.Ptr("global"),
			Owner:        testcommon.AsOwner(rg),
			DataLocation: to.Ptr("UnitedStates"),
		},
	}

	tc.CreateResourceAndWait(commService)

	tc.Expect(commService.Status.Id).ToNot(BeNil())
	armId := *commService.Status.Id

	// Run sub-tests
	tc.RunSubtests(
		testcommon.Subtest{
			Name: "WriteSecrets",
			Test: func(tc *testcommon.KubePerTestContext) {
				CommunicationService_WriteSecrets(tc, commService)
			},
		},
	)

	tc.RunParallelSubtests(
		testcommon.Subtest{
			Name: "Test_EmailService_CRUD",
			Test: func(tc *testcommon.KubePerTestContext) {
				EmailService_CRUD(tc, rg)
			},
		},
	)

	tc.DeleteResourceAndWait(commService)

	// Ensure that the resource was really deleted in Azure
	exists, retryAfter, err := tc.AzureClient.CheckExistenceWithGetByID(tc.Ctx, armId, string(communication.APIVersion_Value))
	tc.Expect(err).ToNot(HaveOccurred())
	tc.Expect(retryAfter).To(BeZero())
	tc.Expect(exists).To(BeFalse())
}

func CommunicationService_WriteSecrets(tc *testcommon.KubePerTestContext, commService *communication.CommunicationService) {
	old := commService.DeepCopy()
	secretName := "communicationservicesecret"
	commService.Spec.OperatorSpec = &communication.CommunicationServiceOperatorSpec{
		Secrets: &communication.CommunicationServiceOperatorSecrets{
			PrimaryKey: &genruntime.SecretDestination{
				Name: secretName,
				Key:  "primaryKey",
			},
			PrimaryConnectionString: &genruntime.SecretDestination{
				Name: secretName,
				Key:  "primaryConnectionString",
			},
			SecondaryKey: &genruntime.SecretDestination{
				Name: secretName,
				Key:  "secondaryKey",
			},
			SecondaryConnectionString: &genruntime.SecretDestination{
				Name: secretName,
				Key:  "secondaryConnectionString",
			},
		},
	}
	tc.PatchResourceAndWait(old, commService)

	tc.ExpectSecretHasKeys(
		secretName,
		"primaryKey",
		"primaryConnectionString",
		"secondaryKey",
		"secondaryConnectionString",
	)
}

func EmailService_CRUD(tc *testcommon.KubePerTestContext, rg *resources.ResourceGroup) {
	emailService := &communication.EmailService{
		ObjectMeta: tc.MakeObjectMeta("emailsvc"),
		Spec: communication.EmailService_Spec{
			Location:     to.Ptr("global"),
			Owner:        testcommon.AsOwner(rg),
			DataLocation: to.Ptr("UnitedStates"),
		},
	}

	tc.CreateResourceAndWait(emailService)

	tc.Expect(emailService.Status.Id).ToNot(BeNil())
	armId := *emailService.Status.Id

	// Test child resources
	tc.RunParallelSubtests(
		testcommon.Subtest{
			Name: "Test_Domain_CRUD",
			Test: func(tc *testcommon.KubePerTestContext) {
				Domain_CRUD(tc, testcommon.AsOwner(emailService))
			},
		},
	)

	tc.DeleteResourceAndWait(emailService)

	// Ensure that the resource was really deleted in Azure
	exists, retryAfter, err := tc.AzureClient.CheckExistenceWithGetByID(tc.Ctx, armId, string(communication.APIVersion_Value))
	tc.Expect(err).ToNot(HaveOccurred())
	tc.Expect(retryAfter).To(BeZero())
	tc.Expect(exists).To(BeFalse())
}

func Domain_CRUD(tc *testcommon.KubePerTestContext, owner *genruntime.KnownResourceReference) {
	domain := &communication.Domain{
		ObjectMeta: tc.MakeObjectMeta("domain"),
		Spec: communication.Domain_Spec{
			Location:         to.Ptr("global"),
			Owner:            owner,
			DomainManagement: to.Ptr(communication.DomainManagement_AzureManaged),
		},
	}

	tc.CreateResourceAndWait(domain)

	tc.Expect(domain.Status.Id).ToNot(BeNil())

	// Test SenderUsername as a child of Domain
	tc.RunSubtests(
		testcommon.Subtest{
			Name: "Test_SenderUsername_CRUD",
			Test: func(tc *testcommon.KubePerTestContext) {
				SenderUsername_CRUD(tc, testcommon.AsOwner(domain))
			},
		},
	)
}

func SenderUsername_CRUD(tc *testcommon.KubePerTestContext, owner *genruntime.KnownResourceReference) {
	senderUsername := &communication.SenderUsername{
		ObjectMeta: tc.MakeObjectMeta("sender"),
		Spec: communication.SenderUsername_Spec{
			Owner:       owner,
			Username:    to.Ptr("testsender"),
			DisplayName: to.Ptr("Test Sender"),
		},
	}

	tc.CreateResourceAndWait(senderUsername)

	tc.Expect(senderUsername.Status.Id).ToNot(BeNil())
}
