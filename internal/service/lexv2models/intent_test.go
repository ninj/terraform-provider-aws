// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lexv2models_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	lextypes "github.com/aws/aws-sdk-go-v2/service/lexmodelsv2/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	tflexv2models "github.com/hashicorp/terraform-provider-aws/internal/service/lexv2models"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// TestIntentAutoFlex is designed to extensively evaluate the capabilities of Intent expansion and
// flattening when utilizing the new feature called "autoflex." Given that autoflex is a recent
// addition and has been built upon the foundation of Intent, these unit tests play a crucial role
// in ensuring the reliability of the implementation.
//
// Looking ahead, for typical scenarios involving straightforward applications of autoflex's Expand
// and Flatten, it is generally unnecessary to conduct tests at the same level of detail as seen in
// this specific autoflex unit test. This guideline is applicable unless dealing with intricate
// resource schemas or situations where there is a genuine concern about the overall functionality.
// In such complex cases, it might still be advisable to perform thorough unit testing with
// autoflex to ensure everything functions as expected.
func TestIntentAutoFlex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testString := "b72d06fd-2b78-5fe2-a6a3-e06e5efde347"
	testString2 := "a47c2004-f58b-5982-880a-f68c80f6307c"

	ssmlMessageTF := tflexv2models.SSMLMessage{
		Value: types.StringValue(testString),
	}
	ssmlMessageAWS := lextypes.SSMLMessage{
		Value: aws.String(testString),
	}

	plainTextMessageTF := tflexv2models.PlainTextMessage{
		Value: types.StringValue(testString),
	}
	plainTextMessageAWS := lextypes.PlainTextMessage{
		Value: aws.String(testString),
	}

	buttonTF := tflexv2models.Button{
		Text:  types.StringValue(testString),
		Value: types.StringValue(testString),
	}
	buttonAWS := lextypes.Button{
		Text:  aws.String(testString),
		Value: aws.String(testString),
	}

	buttonsTF := []tflexv2models.Button{
		buttonTF,
	}
	buttonsAWS := []lextypes.Button{
		buttonAWS,
	}

	imageResponseCardTF := tflexv2models.ImageResponseCard{
		Title:    types.StringValue(testString),
		Button:   fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.Button](ctx, buttonsTF),
		ImageURL: types.StringValue(testString),
		Subtitle: types.StringValue(testString),
	}
	imageResponseCardAWS := lextypes.ImageResponseCard{
		Title:    aws.String(testString),
		Buttons:  buttonsAWS,
		ImageUrl: aws.String(testString),
		Subtitle: aws.String(testString),
	}

	customPayloadTF := tflexv2models.CustomPayload{
		Value: types.StringValue(testString),
	}
	customPayloadAWS := lextypes.CustomPayload{
		Value: aws.String(testString),
	}

	messageTF := tflexv2models.Message{
		CustomPayload:     fwtypes.NewListNestedObjectValueOfPtr(ctx, &customPayloadTF),
		ImageResponseCard: fwtypes.NewListNestedObjectValueOfPtr(ctx, &imageResponseCardTF),
		PlainTextMessage:  fwtypes.NewListNestedObjectValueOfPtr(ctx, &plainTextMessageTF),
		SSMLMessage:       fwtypes.NewListNestedObjectValueOfPtr(ctx, &ssmlMessageTF),
	}
	messageAWS := lextypes.Message{
		CustomPayload:     &customPayloadAWS,
		ImageResponseCard: &imageResponseCardAWS,
		PlainTextMessage:  &plainTextMessageAWS,
		SsmlMessage:       &ssmlMessageAWS,
	}

	messagesTF := []tflexv2models.Message{
		messageTF,
	}
	messagesAWS := []lextypes.Message{
		messageAWS,
	}

	messageGroupTF := tflexv2models.MessageGroup{
		Message:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &messageTF),
		Variations: fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.Message](ctx, messagesTF),
	}
	messageGroupAWS := []lextypes.MessageGroup{
		{
			Message:    &messageAWS,
			Variations: messagesAWS,
		},
	}

	responseSpecificationTF := tflexv2models.ResponseSpecification{
		MessageGroup:   fwtypes.NewListNestedObjectValueOfPtr[tflexv2models.MessageGroup](ctx, &messageGroupTF),
		AllowInterrupt: types.BoolValue(true),
	}
	responseSpecificationAWS := lextypes.ResponseSpecification{
		MessageGroups: []lextypes.MessageGroup{
			{
				Message: &messageAWS,
				Variations: []lextypes.Message{
					messageAWS,
				},
			},
		},
		AllowInterrupt: aws.Bool(true),
	}

	slotValueTF := tflexv2models.SlotValue{
		InterpretedValue: types.StringValue(testString),
	}
	slotValueAWS := lextypes.SlotValue{
		InterpretedValue: aws.String(testString),
	}

	slotValueOverrideAWS := lextypes.SlotValueOverride{
		Shape: lextypes.SlotShapeList,
		Value: &slotValueAWS,
		//Values: fwtypes.NewListNestedObjectValueOfValueSlice(ctx, []tflexv2models.SlotValueOverride{ // recursive so must be defined in line instead of in variable
		//	{
		//		Shape: types.StringValue(testString),
		//		Value: fwtypes.NewListNestedObjectValueOfPtr(ctx, &slotValueTF),
		//	},
		//}
	}

	slotValueOverrideMapTF := tflexv2models.SlotValueOverride{
		MapBlockKey: types.StringValue(testString),
		Shape:       fwtypes.StringEnumValue(lextypes.SlotShapeList),
		Value:       fwtypes.NewListNestedObjectValueOfPtr(ctx, &slotValueTF),
		//Values: fwtypes.NewListNestedObjectValueOfValueSlice(ctx, []tflexv2models.SlotValueOverride{ // recursive so must be defined in line instead of in variable
		//	{
		//		Shape: types.StringValue(testString),
		//		Value: fwtypes.NewListNestedObjectValueOfPtr(ctx, &slotValueTF),
		//	},
		//}
	}
	slotValueOverrideMapAWS := map[string]lextypes.SlotValueOverride{
		testString: slotValueOverrideAWS,
	}

	intentOverrideTF := tflexv2models.IntentOverride{
		Name: types.StringValue(testString),
		Slot: fwtypes.NewListNestedObjectValueOfPtr(ctx, &slotValueOverrideMapTF),
	}
	intentOverrideAWS := lextypes.IntentOverride{
		Name:  aws.String(testString),
		Slots: slotValueOverrideMapAWS,
	}

	dialogActionTF := tflexv2models.DialogAction{
		Type:                fwtypes.StringEnumValue(lextypes.DialogActionTypeCloseIntent),
		SlotToElicit:        types.StringValue(testString),
		SuppressNextMessage: types.BoolValue(true),
	}
	dialogActionAWS := lextypes.DialogAction{
		Type:                lextypes.DialogActionTypeCloseIntent,
		SlotToElicit:        aws.String(testString),
		SuppressNextMessage: aws.Bool(true),
	}

	dialogStateTF := tflexv2models.DialogState{
		DialogAction: fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogActionTF),
		Intent:       fwtypes.NewListNestedObjectValueOfPtr(ctx, &intentOverrideTF),
		SessionAttributes: fwtypes.NewMapValueOf(ctx, map[string]basetypes.StringValue{
			testString: basetypes.NewStringValue(testString2),
		}),
	}
	dialogStateAWS := lextypes.DialogState{
		DialogAction: &dialogActionAWS,
		Intent:       &intentOverrideAWS,
		SessionAttributes: map[string]string{
			testString: testString2,
		},
	}

	conditionTF := tflexv2models.Condition{
		ExpressionString: types.StringValue(testString),
	}

	defaultConditionalBranchTF := tflexv2models.DefaultConditionalBranch{
		NextStep: fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
		Response: fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
	}

	conditionalSpecificationTF := tflexv2models.ConditionalSpecification{
		Active: types.BoolValue(true),
		ConditionalBranch: fwtypes.NewListNestedObjectValueOfValueSlice(ctx, []tflexv2models.ConditionalBranch{{
			Condition: fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionTF),
			Name:      types.StringValue(testString),
			NextStep:  fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
			Response:  fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
		}}),

		DefaultBranch: fwtypes.NewListNestedObjectValueOfPtr(ctx, &defaultConditionalBranchTF),
	}
	conditionalSpecificationAWS := lextypes.ConditionalSpecification{
		Active: aws.Bool(true),
		ConditionalBranches: []lextypes.ConditionalBranch{{
			Condition: &lextypes.Condition{
				ExpressionString: aws.String(testString),
			},
			Name:     aws.String(testString),
			NextStep: &dialogStateAWS,
			Response: &responseSpecificationAWS,
		}},
		DefaultBranch: &lextypes.DefaultConditionalBranch{
			NextStep: &dialogStateAWS,
			Response: &responseSpecificationAWS,
		},
	}

	intentClosingSettingTF := tflexv2models.IntentClosingSetting{
		Active:          types.BoolValue(true),
		ClosingResponse: fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
		Conditional:     fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionalSpecificationTF),
		NextStep:        fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
	}
	intentClosingSettingAWS := lextypes.IntentClosingSetting{
		Active:          aws.Bool(true),
		ClosingResponse: &responseSpecificationAWS,
		Conditional:     &conditionalSpecificationAWS,
		NextStep:        &dialogStateAWS,
	}

	allowedInputTypesTF := tflexv2models.AllowedInputTypes{
		AllowAudioInput: types.BoolValue(true),
		AllowDTMFInput:  types.BoolValue(true),
	}
	allowedInputTypesAWS := lextypes.AllowedInputTypes{
		AllowAudioInput: aws.Bool(true),
		AllowDTMFInput:  aws.Bool(true),
	}

	audioSpecificationTF := tflexv2models.AudioSpecification{
		EndTimeoutMs: types.Int64Value(1),
		MaxLengthMs:  types.Int64Value(1),
	}
	audioSpecificationAWS := lextypes.AudioSpecification{
		EndTimeoutMs: aws.Int32(1),
		MaxLengthMs:  aws.Int32(1),
	}

	dtmfSpecificationTF := tflexv2models.DTMFSpecification{
		DeletionCharacter: types.StringValue(testString),
		EndCharacter:      types.StringValue(testString),
		EndTimeoutMs:      types.Int64Value(1),
		MaxLength:         types.Int64Value(1),
	}
	dtmfSpecificationAWS := lextypes.DTMFSpecification{
		DeletionCharacter: aws.String(testString),
		EndCharacter:      aws.String(testString),
		EndTimeoutMs:      aws.Int32(1),
		MaxLength:         aws.Int32(1),
	}

	audioAndDTMFInputSpecificationTF := tflexv2models.AudioAndDTMFInputSpecification{
		StartTimeoutMs:     types.Int64Value(1),
		AudioSpecification: fwtypes.NewListNestedObjectValueOfPtr(ctx, &audioSpecificationTF),
		DTMFSpecification:  fwtypes.NewListNestedObjectValueOfPtr(ctx, &dtmfSpecificationTF),
	}
	audioAndDTMFInputSpecificationAWS := lextypes.AudioAndDTMFInputSpecification{
		StartTimeoutMs:     aws.Int32(1),
		AudioSpecification: &audioSpecificationAWS,
		DtmfSpecification:  &dtmfSpecificationAWS,
	}

	textInputSpecificationTF := tflexv2models.TextInputSpecification{
		StartTimeoutMs: types.Int64Value(1),
	}
	textInputSpecificationAWS := lextypes.TextInputSpecification{
		StartTimeoutMs: aws.Int32(1),
	}

	promptAttemptSpecificationTF := tflexv2models.PromptAttemptsSpecification{
		MapBlockKey:                    fwtypes.StringEnumValue(tflexv2models.PromptAttemptsTypeInitial),
		AllowedInputTypes:              fwtypes.NewListNestedObjectValueOfPtr(ctx, &allowedInputTypesTF),
		AllowInterrupt:                 types.BoolValue(true),
		AudioAndDTMFInputSpecification: fwtypes.NewListNestedObjectValueOfPtr(ctx, &audioAndDTMFInputSpecificationTF),
		TextInputSpecification:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &textInputSpecificationTF),
	}
	promptAttemptSpecificationAWS := lextypes.PromptAttemptSpecification{
		AllowedInputTypes:              &allowedInputTypesAWS,
		AllowInterrupt:                 aws.Bool(true),
		AudioAndDTMFInputSpecification: &audioAndDTMFInputSpecificationAWS,
		TextInputSpecification:         &textInputSpecificationAWS,
	}

	promptSpecificationTF := tflexv2models.PromptSpecification{
		MaxRetries:                  types.Int64Value(1),
		MessageGroup:                fwtypes.NewListNestedObjectValueOfPtr(ctx, &messageGroupTF),
		AllowInterrupt:              types.BoolValue(true),
		MessageSelectionStrategy:    fwtypes.StringEnumValue(lextypes.MessageSelectionStrategyOrdered),
		PromptAttemptsSpecification: fwtypes.NewListNestedObjectValueOfPtr(ctx, &promptAttemptSpecificationTF),
	}
	promptSpecificationAWS := lextypes.PromptSpecification{
		MaxRetries:               aws.Int32(1),
		MessageGroups:            messageGroupAWS,
		AllowInterrupt:           aws.Bool(true),
		MessageSelectionStrategy: lextypes.MessageSelectionStrategyOrdered,
		PromptAttemptsSpecification: map[string]lextypes.PromptAttemptSpecification{
			string(tflexv2models.PromptAttemptsTypeInitial): promptAttemptSpecificationAWS,
		},
	}

	failureSuccessTimeoutTF := tflexv2models.FailureSuccessTimeout{
		FailureConditional: fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionalSpecificationTF),
		FailureNextStep:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
		FailureResponse:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
		SuccessConditional: fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionalSpecificationTF),
		SuccessNextStep:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
		SuccessResponse:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
		TimeoutConditional: fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionalSpecificationTF),
		TimeoutNextStep:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
		TimeoutResponse:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
	}

	postCodeHookSpecificationAWS := lextypes.PostDialogCodeHookInvocationSpecification{
		FailureConditional: &conditionalSpecificationAWS,
		FailureNextStep:    &dialogStateAWS,
		FailureResponse:    &responseSpecificationAWS,
		SuccessConditional: &conditionalSpecificationAWS,
		SuccessNextStep:    &dialogStateAWS,
		SuccessResponse:    &responseSpecificationAWS,
		TimeoutConditional: &conditionalSpecificationAWS,
		TimeoutNextStep:    &dialogStateAWS,
		TimeoutResponse:    &responseSpecificationAWS,
	}
	postFulfillmentStatusSpecificationAWS := lextypes.PostFulfillmentStatusSpecification{
		FailureConditional: &conditionalSpecificationAWS,
		FailureNextStep:    &dialogStateAWS,
		FailureResponse:    &responseSpecificationAWS,
		SuccessConditional: &conditionalSpecificationAWS,
		SuccessNextStep:    &dialogStateAWS,
		SuccessResponse:    &responseSpecificationAWS,
		TimeoutConditional: &conditionalSpecificationAWS,
		TimeoutNextStep:    &dialogStateAWS,
		TimeoutResponse:    &responseSpecificationAWS,
	}

	dialogCodeHookInvocationSettingTF := tflexv2models.DialogCodeHookInvocationSetting{
		Active:                    types.BoolValue(true),
		EnableCodeHookInvocation:  types.BoolValue(true),
		InvocationLabel:           types.StringValue(testString),
		PostCodeHookSpecification: fwtypes.NewListNestedObjectValueOfPtr(ctx, &failureSuccessTimeoutTF),
	}
	dialogCodeHookInvocationSettingAWS := lextypes.DialogCodeHookInvocationSetting{
		Active:                    aws.Bool(true),
		EnableCodeHookInvocation:  aws.Bool(true),
		InvocationLabel:           aws.String(testString),
		PostCodeHookSpecification: &postCodeHookSpecificationAWS,
	}

	elicitationCodeHookTF := tflexv2models.ElicitationCodeHookInvocationSetting{
		EnableCodeHookInvocation: types.BoolValue(true),
		InvocationLabel:          types.StringValue(testString),
	}
	elicitationCodeHookAWS := lextypes.ElicitationCodeHookInvocationSetting{
		EnableCodeHookInvocation: aws.Bool(true),
		InvocationLabel:          aws.String(testString),
	}

	intentConfirmationSettingTF := tflexv2models.IntentConfirmationSetting{
		PromptSpecification:     fwtypes.NewListNestedObjectValueOfPtr(ctx, &promptSpecificationTF),
		Active:                  types.BoolValue(true),
		CodeHook:                fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogCodeHookInvocationSettingTF),
		ConfirmationConditional: fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionalSpecificationTF),
		ConfirmationNextStep:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
		ConfirmationResponse:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
		DeclinationConditional:  fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionalSpecificationTF),
		DeclinationNextStep:     fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
		DeclinationResponse:     fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
		ElicitationCodeHook:     fwtypes.NewListNestedObjectValueOfPtr(ctx, &elicitationCodeHookTF),
		FailureConditional:      fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionalSpecificationTF),
		FailureNextStep:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
		FailureResponse:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
	}
	intentConfirmationSettingAWS := lextypes.IntentConfirmationSetting{
		PromptSpecification:     &promptSpecificationAWS,
		Active:                  aws.Bool(true),
		CodeHook:                &dialogCodeHookInvocationSettingAWS,
		ConfirmationConditional: &conditionalSpecificationAWS,
		ConfirmationNextStep:    &dialogStateAWS,
		ConfirmationResponse:    &responseSpecificationAWS,
		DeclinationConditional:  &conditionalSpecificationAWS,
		DeclinationNextStep:     &dialogStateAWS,
		DeclinationResponse:     &responseSpecificationAWS,
		ElicitationCodeHook:     &elicitationCodeHookAWS,
		FailureConditional:      &conditionalSpecificationAWS,
		FailureNextStep:         &dialogStateAWS,
		FailureResponse:         &responseSpecificationAWS,
	}

	dialogCodeHookSettingsTF := tflexv2models.DialogCodeHookSettings{
		Enabled: types.BoolValue(true),
	}
	dialogCodeHookSettingsAWS := lextypes.DialogCodeHookSettings{
		Enabled: true,
	}

	fulfillmentStartResponseSpecificationTF := tflexv2models.FulfillmentStartResponseSpecification{
		DelayInSeconds: types.Int64Value(1),
		MessageGroup:   fwtypes.NewListNestedObjectValueOfPtr(ctx, &messageGroupTF),
		AllowInterrupt: types.BoolValue(true),
	}
	fulfillmentStartResponseSpecificationAWS := lextypes.FulfillmentStartResponseSpecification{
		DelayInSeconds: aws.Int32(1),
		MessageGroups:  messageGroupAWS,
		AllowInterrupt: aws.Bool(true),
	}

	fulfillmentUpdateResponseSpecificationTF := tflexv2models.FulfillmentUpdateResponseSpecification{
		FrequencyInSeconds: types.Int64Value(1),
		MessageGroup:       fwtypes.NewListNestedObjectValueOfPtr(ctx, &messageGroupTF),
		AllowInterrupt:     types.BoolValue(true),
	}
	fulfillmentUpdateResponseSpecificationAWS := lextypes.FulfillmentUpdateResponseSpecification{
		FrequencyInSeconds: aws.Int32(1),
		MessageGroups:      messageGroupAWS,
		AllowInterrupt:     aws.Bool(true),
	}

	fulfillmentUpdatesSpecificationTF := tflexv2models.FulfillmentUpdatesSpecification{
		Active:           types.BoolValue(true),
		StartResponse:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &fulfillmentStartResponseSpecificationTF),
		TimeoutInSeconds: types.Int64Value(1),
		UpdateResponse:   fwtypes.NewListNestedObjectValueOfPtr(ctx, &fulfillmentUpdateResponseSpecificationTF),
	}
	fulfillmentUpdatesSpecificationAWS := lextypes.FulfillmentUpdatesSpecification{
		Active:           aws.Bool(true),
		StartResponse:    &fulfillmentStartResponseSpecificationAWS,
		TimeoutInSeconds: aws.Int32(1),
		UpdateResponse:   &fulfillmentUpdateResponseSpecificationAWS,
	}

	fulfillmentCodeHookSettingsTF := tflexv2models.FulfillmentCodeHookSettings{
		Enabled:                            types.BoolValue(true),
		Active:                             types.BoolValue(true),
		FulfillmentUpdatesSpecification:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &fulfillmentUpdatesSpecificationTF),
		PostFulfillmentStatusSpecification: fwtypes.NewListNestedObjectValueOfPtr(ctx, &failureSuccessTimeoutTF),
	}
	fulfillmentCodeHookSettingsAWS := lextypes.FulfillmentCodeHookSettings{
		Enabled:                            true,
		Active:                             aws.Bool(true),
		FulfillmentUpdatesSpecification:    &fulfillmentUpdatesSpecificationAWS,
		PostFulfillmentStatusSpecification: &postFulfillmentStatusSpecificationAWS,
	}

	initialResponseSettingTF := tflexv2models.InitialResponseSetting{
		CodeHook:        fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogCodeHookInvocationSettingTF),
		Conditional:     fwtypes.NewListNestedObjectValueOfPtr(ctx, &conditionalSpecificationTF),
		InitialResponse: fwtypes.NewListNestedObjectValueOfPtr(ctx, &responseSpecificationTF),
		NextStep:        fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogStateTF),
	}
	initialResponseSettingAWS := lextypes.InitialResponseSetting{
		CodeHook:        &dialogCodeHookInvocationSettingAWS,
		Conditional:     &conditionalSpecificationAWS,
		InitialResponse: &responseSpecificationAWS,
		NextStep:        &dialogStateAWS,
	}

	inputContextTF := tflexv2models.InputContext{
		Name: types.StringValue(testString),
	}
	inputContextAWS := lextypes.InputContext{
		Name: aws.String(testString),
	}

	inputContextsTF := []tflexv2models.InputContext{
		inputContextTF,
	}
	inputContextsAWS := []lextypes.InputContext{
		inputContextAWS,
	}

	kendraConfigurationTF := tflexv2models.KendraConfiguration{
		KendraIndex:              types.StringValue(testString),
		QueryFilterString:        types.StringValue(testString),
		QueryFilterStringEnabled: types.BoolValue(true),
	}
	kendraConfigurationAWS := lextypes.KendraConfiguration{
		KendraIndex:              aws.String(testString),
		QueryFilterString:        aws.String(testString),
		QueryFilterStringEnabled: true,
	}

	outputContextTF := tflexv2models.OutputContext{
		Name:                types.StringValue(testString),
		TimeToLiveInSeconds: types.Int64Value(1),
		TurnsToLive:         types.Int64Value(1),
	}
	outputContextAWS := lextypes.OutputContext{
		Name:                aws.String(testString),
		TimeToLiveInSeconds: aws.Int32(1),
		TurnsToLive:         aws.Int32(1),
	}

	outputContextsTF := []tflexv2models.OutputContext{
		outputContextTF,
	}
	outputContextsAWS := []lextypes.OutputContext{
		outputContextAWS,
	}

	sampleUtteranceTF := tflexv2models.SampleUtterance{
		Utterance: types.StringValue(testString),
	}
	sampleUtteranceAWS := lextypes.SampleUtterance{
		Utterance: aws.String(testString),
	}

	sampleUtterancesTF := []tflexv2models.SampleUtterance{
		sampleUtteranceTF,
	}
	sampleUtterancesAWS := []lextypes.SampleUtterance{
		sampleUtteranceAWS,
	}

	slotPriorityTF := tflexv2models.SlotPriority{
		Priority: types.Int64Value(1),
		SlotID:   types.StringValue(testString),
	}
	slotPriorityAWS := lextypes.SlotPriority{
		Priority: aws.Int32(1),
		SlotId:   aws.String(testString),
	}

	slotPrioritiesTF := []tflexv2models.SlotPriority{
		slotPriorityTF,
	}
	slotPrioritiesAWS := []lextypes.SlotPriority{
		slotPriorityAWS,
	}

	intentCreateTF := tflexv2models.ResourceIntentData{
		BotID:                  types.StringValue(testString),
		BotVersion:             types.StringValue(testString),
		Name:                   types.StringValue(testString),
		LocaleID:               types.StringValue(testString),
		Description:            types.StringValue(testString),
		DialogCodeHook:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogCodeHookSettingsTF),
		FulfillmentCodeHook:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &fulfillmentCodeHookSettingsTF),
		InitialResponseSetting: fwtypes.NewListNestedObjectValueOfPtr(ctx, &initialResponseSettingTF),
		InputContext:           fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.InputContext](ctx, inputContextsTF),
		ClosingSetting:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &intentClosingSettingTF),
		ConfirmationSetting:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &intentConfirmationSettingTF),
		KendraConfiguration:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &kendraConfigurationTF),
		OutputContext:          fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.OutputContext](ctx, outputContextsTF),
		ParentIntentSignature:  types.StringValue(testString),
		SampleUtterance:        fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.SampleUtterance](ctx, sampleUtterancesTF),
	}
	intentCreateAWS := lexmodelsv2.CreateIntentInput{
		BotId:                     aws.String(testString),
		BotVersion:                aws.String(testString),
		IntentName:                aws.String(testString),
		LocaleId:                  aws.String(testString),
		Description:               aws.String(testString),
		DialogCodeHook:            &dialogCodeHookSettingsAWS,
		FulfillmentCodeHook:       &fulfillmentCodeHookSettingsAWS,
		InitialResponseSetting:    &initialResponseSettingAWS,
		InputContexts:             inputContextsAWS,
		IntentClosingSetting:      &intentClosingSettingAWS,
		IntentConfirmationSetting: &intentConfirmationSettingAWS,
		KendraConfiguration:       &kendraConfigurationAWS,
		OutputContexts:            outputContextsAWS,
		ParentIntentSignature:     aws.String(testString),
		SampleUtterances:          sampleUtterancesAWS,
	}

	intentModifyTF := tflexv2models.ResourceIntentData{
		BotID:                  types.StringValue(testString),
		BotVersion:             types.StringValue(testString),
		Name:                   types.StringValue(testString),
		LocaleID:               types.StringValue(testString),
		Description:            types.StringValue(testString),
		DialogCodeHook:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogCodeHookSettingsTF),
		FulfillmentCodeHook:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &fulfillmentCodeHookSettingsTF),
		InitialResponseSetting: fwtypes.NewListNestedObjectValueOfPtr(ctx, &initialResponseSettingTF),
		InputContext:           fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.InputContext](ctx, inputContextsTF),
		ClosingSetting:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &intentClosingSettingTF),
		ConfirmationSetting:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &intentConfirmationSettingTF),
		KendraConfiguration:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &kendraConfigurationTF),
		OutputContext:          fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.OutputContext](ctx, outputContextsTF),
		ParentIntentSignature:  types.StringValue(testString),
		SampleUtterance:        fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.SampleUtterance](ctx, sampleUtterancesTF),
		SlotPriority:           fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.SlotPriority](ctx, slotPrioritiesTF),
	}
	intentModifyAWS := lexmodelsv2.UpdateIntentInput{
		BotId:                     aws.String(testString),
		BotVersion:                aws.String(testString),
		IntentName:                aws.String(testString),
		LocaleId:                  aws.String(testString),
		Description:               aws.String(testString),
		DialogCodeHook:            &dialogCodeHookSettingsAWS,
		FulfillmentCodeHook:       &fulfillmentCodeHookSettingsAWS,
		InitialResponseSetting:    &initialResponseSettingAWS,
		InputContexts:             inputContextsAWS,
		IntentClosingSetting:      &intentClosingSettingAWS,
		IntentConfirmationSetting: &intentConfirmationSettingAWS,
		KendraConfiguration:       &kendraConfigurationAWS,
		OutputContexts:            outputContextsAWS,
		ParentIntentSignature:     aws.String(testString),
		SampleUtterances:          sampleUtterancesAWS,
		SlotPriorities:            slotPrioritiesAWS,
	}

	testTimeStr := "2023-12-08T09:34:01Z"
	testTimeTime := errs.Must(time.Parse(time.RFC3339, testTimeStr))

	intentDescribeTF := tflexv2models.ResourceIntentData{
		BotID:                  types.StringValue(testString),
		BotVersion:             types.StringValue(testString),
		ClosingSetting:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &intentClosingSettingTF),
		ConfirmationSetting:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &intentConfirmationSettingTF),
		CreationDateTime:       fwtypes.TimestampValue(testTimeStr),
		Description:            types.StringValue(testString),
		DialogCodeHook:         fwtypes.NewListNestedObjectValueOfPtr(ctx, &dialogCodeHookSettingsTF),
		FulfillmentCodeHook:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &fulfillmentCodeHookSettingsTF),
		IntentID:               types.StringValue(testString),
		InitialResponseSetting: fwtypes.NewListNestedObjectValueOfPtr(ctx, &initialResponseSettingTF),
		InputContext:           fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.InputContext](ctx, inputContextsTF),
		KendraConfiguration:    fwtypes.NewListNestedObjectValueOfPtr(ctx, &kendraConfigurationTF),
		LastUpdatedDateTime:    fwtypes.TimestampValue(testTimeStr),
		LocaleID:               types.StringValue(testString),
		Name:                   types.StringValue(testString),
		OutputContext:          fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.OutputContext](ctx, outputContextsTF),
		ParentIntentSignature:  types.StringValue(testString),
		SampleUtterance:        fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.SampleUtterance](ctx, sampleUtterancesTF),
		SlotPriority:           fwtypes.NewListNestedObjectValueOfValueSlice[tflexv2models.SlotPriority](ctx, slotPrioritiesTF),
	}
	intentDescribeAWS := lexmodelsv2.DescribeIntentOutput{
		BotId:                     aws.String(testString),
		BotVersion:                aws.String(testString),
		CreationDateTime:          aws.Time(testTimeTime),
		Description:               aws.String(testString),
		DialogCodeHook:            &dialogCodeHookSettingsAWS,
		FulfillmentCodeHook:       &fulfillmentCodeHookSettingsAWS,
		InitialResponseSetting:    &initialResponseSettingAWS,
		InputContexts:             inputContextsAWS,
		IntentClosingSetting:      &intentClosingSettingAWS,
		IntentConfirmationSetting: &intentConfirmationSettingAWS,
		IntentId:                  aws.String(testString),
		IntentName:                aws.String(testString),
		KendraConfiguration:       &kendraConfigurationAWS,
		LastUpdatedDateTime:       aws.Time(testTimeTime),
		LocaleId:                  aws.String(testString),
		OutputContexts:            outputContextsAWS,
		ParentIntentSignature:     aws.String(testString),
		SampleUtterances:          sampleUtterancesAWS,
		SlotPriorities:            slotPrioritiesAWS,
	}

	testCases := []struct {
		TestName string
		TFFull   any
		AWSFull  any
		TFEmpty  any
		AWSEmpty any
		WantErr  bool
	}{
		{
			TestName: "message",
			TFFull:   &messageTF,
			TFEmpty:  &tflexv2models.Message{},
			AWSFull:  &messageAWS,
			AWSEmpty: &lextypes.Message{},
		},
		{
			TestName: "responseSpecification",
			TFFull:   &responseSpecificationTF,
			TFEmpty:  &tflexv2models.ResponseSpecification{},
			AWSFull:  &responseSpecificationAWS,
			AWSEmpty: &lextypes.ResponseSpecification{},
		},
		{
			TestName: "promptSpecification",
			TFFull:   &promptSpecificationTF,
			TFEmpty:  &tflexv2models.PromptSpecification{},
			AWSFull:  &promptSpecificationAWS,
			AWSEmpty: &lextypes.PromptSpecification{},
		},
		{
			TestName: "dialogState",
			TFFull:   &dialogStateTF,
			TFEmpty:  &tflexv2models.DialogState{},
			AWSFull:  &dialogStateAWS,
			AWSEmpty: &lextypes.DialogState{},
		},
		{
			TestName: "dialogAction",
			TFFull:   &dialogActionTF,
			TFEmpty:  &tflexv2models.DialogAction{},
			AWSFull:  &dialogActionAWS,
			AWSEmpty: &lextypes.DialogAction{},
		},
		{
			TestName: "conditionalSpecification",
			TFFull:   &conditionalSpecificationTF,
			TFEmpty:  &tflexv2models.ConditionalSpecification{},
			AWSFull:  &conditionalSpecificationAWS,
			AWSEmpty: &lextypes.ConditionalSpecification{},
		},
		{
			TestName: "intentClosingSetting",
			TFFull:   &intentClosingSettingTF,
			TFEmpty:  &tflexv2models.IntentClosingSetting{},
			AWSFull:  &intentClosingSettingAWS,
			AWSEmpty: &lextypes.IntentClosingSetting{},
		},
		{
			TestName: "intentConfirmationSetting",
			TFFull:   &intentConfirmationSettingTF,
			TFEmpty:  &tflexv2models.IntentConfirmationSetting{},
			AWSFull:  &intentConfirmationSettingAWS,
			AWSEmpty: &lextypes.IntentConfirmationSetting{},
		},
		{
			TestName: "intentOverride",
			TFFull:   &intentOverrideTF,
			TFEmpty:  &tflexv2models.IntentOverride{},
			AWSFull:  &intentOverrideAWS,
			AWSEmpty: &lextypes.IntentOverride{},
		},
		{
			TestName: "slotValue",
			TFFull:   &slotValueTF,
			TFEmpty:  &tflexv2models.SlotValue{},
			AWSFull:  &slotValueAWS,
			AWSEmpty: &lextypes.SlotValue{},
		},
		{
			TestName: "create intent",
			TFFull:   &intentCreateTF,
			TFEmpty:  &tflexv2models.ResourceIntentData{},
			AWSFull:  &intentCreateAWS,
			AWSEmpty: &lexmodelsv2.CreateIntentInput{},
		},
		{
			TestName: "update intent",
			TFFull:   &intentModifyTF,
			TFEmpty:  &tflexv2models.ResourceIntentData{},
			AWSFull:  &intentModifyAWS,
			AWSEmpty: &lexmodelsv2.UpdateIntentInput{},
		},
		{
			TestName: "describe intent",
			TFFull:   &intentDescribeTF,
			TFEmpty:  &tflexv2models.ResourceIntentData{},
			AWSFull:  &intentDescribeAWS,
			AWSEmpty: &lexmodelsv2.DescribeIntentOutput{},
		},
	}

	ignoreExpoOpts := cmpopts.IgnoreUnexported(
		lexmodelsv2.CreateIntentInput{},
		lexmodelsv2.DescribeIntentOutput{},
		lexmodelsv2.UpdateIntentInput{},
		lextypes.AllowedInputTypes{},
		lextypes.AudioAndDTMFInputSpecification{},
		lextypes.AudioSpecification{},
		lextypes.Button{},
		lextypes.Condition{},
		lextypes.ConditionalBranch{},
		lextypes.ConditionalSpecification{},
		lextypes.CustomPayload{},
		lextypes.DefaultConditionalBranch{},
		lextypes.DialogAction{},
		lextypes.DialogCodeHookInvocationSetting{},
		lextypes.DialogCodeHookSettings{},
		lextypes.DialogState{},
		lextypes.DTMFSpecification{},
		lextypes.ElicitationCodeHookInvocationSetting{},
		lextypes.FulfillmentCodeHookSettings{},
		lextypes.FulfillmentStartResponseSpecification{},
		lextypes.FulfillmentUpdateResponseSpecification{},
		lextypes.FulfillmentUpdatesSpecification{},
		lextypes.ImageResponseCard{},
		lextypes.InitialResponseSetting{},
		lextypes.InputContext{},
		lextypes.IntentClosingSetting{},
		lextypes.IntentConfirmationSetting{},
		lextypes.IntentOverride{},
		lextypes.KendraConfiguration{},
		lextypes.Message{},
		lextypes.MessageGroup{},
		lextypes.OutputContext{},
		lextypes.PlainTextMessage{},
		lextypes.PostDialogCodeHookInvocationSpecification{},
		lextypes.PostFulfillmentStatusSpecification{},
		lextypes.PromptAttemptSpecification{},
		lextypes.PromptSpecification{},
		lextypes.ResponseSpecification{},
		lextypes.SampleUtterance{},
		lextypes.SlotPriority{},
		lextypes.SlotValue{},
		lextypes.SlotValueOverride{},
		lextypes.SSMLMessage{},
		lextypes.TextInputSpecification{},
		middleware.Metadata{},
	)

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(fmt.Sprintf("expand %s", testCase.TestName), func(t *testing.T) {
			t.Parallel()

			diags := flex.Expand(context.WithValue(ctx, flex.ResourcePrefix, "Intent"), testCase.TFFull, testCase.AWSEmpty)

			gotErr := diags != nil

			if gotErr != testCase.WantErr {
				t.Errorf("gotErr = %v, wantErr = %v", gotErr, testCase.WantErr)
			}

			if gotErr {
				if !testCase.WantErr {
					t.Errorf("err = %q", diags)
				}
			} else {
				if diff := cmp.Diff(testCase.AWSEmpty, testCase.AWSFull, ignoreExpoOpts); diff != "" {
					t.Errorf("unexpected diff (+wanted, -got): %s", diff)
				}
			}
		})

		t.Run(fmt.Sprintf("flatten %s", testCase.TestName), func(t *testing.T) {
			t.Parallel()

			diags := flex.Flatten(context.WithValue(ctx, flex.ResourcePrefix, "Intent"), testCase.AWSFull, testCase.TFEmpty)

			gotErr := diags != nil

			if gotErr != testCase.WantErr {
				t.Errorf("gotErr = %v, wantErr = %v", gotErr, testCase.WantErr)
			}

			if gotErr {
				if !testCase.WantErr {
					t.Errorf("err = %q", diags)
				}
			} else {
				// because TF type has .Equal method, cmp can act strangely - string comparison shortcut
				// avoids
				if fmt.Sprint(testCase.TFEmpty) == fmt.Sprint(testCase.TFFull) {
					return
				}

				if diff := cmp.Diff(testCase.TFEmpty, testCase.TFFull, ignoreExpoOpts); diff != "" {
					t.Errorf("unexpected diff (+wanted, -got): %s", diff)
				}
			}
		})
	}
}

// Acceptance tests access AWS and cost money to run.

func TestAccLexV2ModelsIntent_basic(t *testing.T) {
	ctx := acctest.Context(t)

	var intent lexmodelsv2.DescribeIntentOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_lexv2models_intent.test"
	botLocaleName := "aws_lexv2models_bot_locale.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.LexV2ModelsEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.LexV2ModelsEndpointID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckIntentDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccIntentConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntentExists(ctx, resourceName, &intent),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrPair(resourceName, "bot_id", botLocaleName, "bot_id"),
					resource.TestCheckResourceAttrPair(resourceName, "bot_version", botLocaleName, "bot_version"),
					resource.TestCheckResourceAttrPair(resourceName, "locale_id", botLocaleName, "locale_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLexV2ModelsIntent_disappears(t *testing.T) {
	ctx := acctest.Context(t)

	var intent lexmodelsv2.DescribeIntentOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_lexv2models_intent.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.LexV2ModelsEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.LexV2ModelsEndpointID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckIntentDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccIntentConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntentExists(ctx, resourceName, &intent),
					acctest.CheckFrameworkResourceDisappears(ctx, acctest.Provider, tflexv2models.ResourceIntent, resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccLexV2ModelsIntent_update(t *testing.T) {
	ctx := acctest.Context(t)

	var intent lexmodelsv2.DescribeIntentOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_lexv2models_intent.test"
	botLocaleName := "aws_lexv2models_bot_locale.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.LexV2ModelsEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.LexV2ModelsEndpointID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckIntentDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccIntentConfig_update(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntentExists(ctx, resourceName, &intent),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrPair(resourceName, "bot_id", botLocaleName, "bot_id"),
					resource.TestCheckResourceAttrPair(resourceName, "bot_version", botLocaleName, "bot_version"),
					resource.TestCheckResourceAttrPair(resourceName, "locale_id", botLocaleName, "locale_id"),
				),
			},
		},
	})
}

func testAccCheckIntentDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).LexV2ModelsClient(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_lexv2models_intent" {
				continue
			}

			_, err := conn.DescribeIntent(ctx, &lexmodelsv2.DescribeIntentInput{
				IntentId:   aws.String(rs.Primary.Attributes["intent_id"]),
				BotId:      aws.String(rs.Primary.Attributes["bot_id"]),
				BotVersion: aws.String(rs.Primary.Attributes["bot_version"]),
				LocaleId:   aws.String(rs.Primary.Attributes["locale_id"]),
			})
			if errs.IsA[*lextypes.ResourceNotFoundException](err) {
				return nil
			}
			if err != nil {
				return create.Error(names.LexV2Models, create.ErrActionCheckingDestroyed, tflexv2models.ResNameIntent, rs.Primary.ID, err)
			}

			return create.Error(names.LexV2Models, create.ErrActionCheckingDestroyed, tflexv2models.ResNameIntent, rs.Primary.ID, errors.New("not destroyed"))
		}

		return nil
	}
}

func testAccCheckIntentExists(ctx context.Context, name string, intent *lexmodelsv2.DescribeIntentOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return create.Error(names.LexV2Models, create.ErrActionCheckingExistence, tflexv2models.ResNameIntent, name, errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return create.Error(names.LexV2Models, create.ErrActionCheckingExistence, tflexv2models.ResNameIntent, name, errors.New("not set"))
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).LexV2ModelsClient(ctx)
		resp, err := conn.DescribeIntent(ctx, &lexmodelsv2.DescribeIntentInput{
			IntentId:   aws.String(rs.Primary.Attributes["intent_id"]),
			BotId:      aws.String(rs.Primary.Attributes["bot_id"]),
			BotVersion: aws.String(rs.Primary.Attributes["bot_version"]),
			LocaleId:   aws.String(rs.Primary.Attributes["locale_id"]),
		})

		if err != nil {
			return create.Error(names.LexV2Models, create.ErrActionCheckingExistence, tflexv2models.ResNameIntent, rs.Primary.ID, err)
		}

		*intent = *resp

		return nil
	}
}

func testAccIntentConfig_base(rName string, ttl int, dp bool) string {
	return fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "lexv2.amazonaws.com"
        }
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "test" {
  role       = aws_iam_role.test.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonLexFullAccess"
}

resource "aws_lexv2models_bot" "test" {
  name                        = %[1]q
  idle_session_ttl_in_seconds = %[2]d
  role_arn                    = aws_iam_role.test.arn

  data_privacy {
    child_directed = %[3]t
  }
}

resource "aws_lexv2models_bot_locale" "test" {
  locale_id                        = "en_US"
  bot_id                           = aws_lexv2models_bot.test.id
  bot_version                      = "DRAFT"
  n_lu_intent_confidence_threshold = 0.7
}

resource "aws_lexv2models_bot_version" "test" {
  bot_id = aws_lexv2models_bot.test.id
  locale_specification = {
    (aws_lexv2models_bot_locale.test.locale_id) = {
      source_bot_version = "DRAFT"
    }
  }
}
`, rName, ttl, dp)
}

func testAccIntentConfig_basic(rName string) string {
	return acctest.ConfigCompose(
		testAccIntentConfig_base(rName, 60, true),
		fmt.Sprintf(`
resource "aws_lexv2models_intent" "test" {
  bot_id      = aws_lexv2models_bot.test.id
  bot_version = aws_lexv2models_bot_locale.test.bot_version
  name        = %[1]q
  locale_id   = aws_lexv2models_bot_locale.test.locale_id
}
`, rName))
}

func testAccIntentConfig_update(rName string) string {
	return acctest.ConfigCompose(
		testAccIntentConfig_base(rName, 60, true),
		fmt.Sprintf(`
resource "aws_lexv2models_intent" "test" {
  bot_id      = aws_lexv2models_bot.test.id
  bot_version = aws_lexv2models_bot_locale.test.bot_version
  name        = %[1]q
  locale_id   = aws_lexv2models_bot_locale.test.locale_id

  confirmation_setting {
    active = true
    
    prompt_specification {
      allow_interrupt = true
      max_retries     = 1
      message_selection_strategy = "Ordered"

      prompt_attempts_specification {
        map_block_key = "Initial"
        
        allowed_input_types {
          allow_audio_input = true
          allow_dtmf_input  = true
        }

        allow_interrupt = true

        audio_and_dtmf_input_specification {
          start_timeout_ms = 1

          audio_specification {
            end_timeout_ms = 1
            max_length_ms = 1
          }

          dtmf_specification {
            deletion_character = "#"
            end_character      = "#"
            end_timeout_ms     = 1
            max_length         = 1
          }
        }

        text_input_specification {
          start_timeout_ms = 1
        }
      
      }
    }
  }
}
`, rName))
}
